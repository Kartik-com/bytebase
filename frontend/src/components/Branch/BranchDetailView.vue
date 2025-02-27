<template>
  <div class="space-y-3 w-full overflow-x-auto px-4 pt-1" v-bind="$attrs">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="w-full flex flex-row justify-start items-center gap-x-2">
        <NInput
          v-model:value="state.schemaDesignTitle"
          class="!w-auto"
          :passively-activated="true"
          :style="titleInputStyle"
          :readonly="!state.isEditingTitle"
          :placeholder="'feature/add-billing'"
          @focus="state.isEditingTitle = true"
          @blur="handleBranchTitleInputBlur"
        />
        <NTag v-if="parentBranch" round>
          {{ $t("schema-designer.parent-branch") }}:
          {{ parentBranch.title }}
        </NTag>
      </div>
      <div>
        <div class="w-full flex flex-row justify-between items-center">
          <div
            v-if="!viewMode"
            class="flex flex-row justify-end items-center space-x-2"
          >
            <template v-if="!state.isEditing">
              <NButton @click="handleEdit">{{ $t("common.edit") }}</NButton>
              <NButton @click="handleMergeBranch">{{
                $t("schema-designer.merge-branch")
              }}</NButton>
              <NButton type="primary" @click="handleApplySchemaDesignClick">{{
                $t("schema-designer.apply-to-database")
              }}</NButton>
            </template>
            <template v-else>
              <NButton @click="handleCancelEdit">{{
                $t("common.cancel")
              }}</NButton>
              <NButton type="primary" @click="handleSaveBranch">{{
                $t("common.save")
              }}</NButton>
            </template>
          </div>
        </div>
      </div>
    </div>

    <NDivider />

    <div
      class="w-full flex flex-row justify-between items-center text-sm mt-1 gap-4"
    >
      <div class="flex flex-row justify-start items-center opacity-80">
        <span class="mr-4 shrink-0"
          >{{ $t("schema-designer.baseline-version") }}:</span
        >
        <DatabaseInfo
          class="flex-nowrap mr-4 shrink-0"
          :database="baselineDatabase"
        />
        <div class="shrink-0 flex-nowrap">
          <NTooltip v-if="changeHistory" trigger="hover">
            <template #trigger> @{{ changeHistory.version }} </template>
            <div class="w-full flex flex-row justify-start items-center">
              <span class="block pr-2 w-full max-w-[32rem] truncate">
                {{ changeHistory.version }} -
                {{ changeHistory.description }}
              </span>
              <span class="opacity-60">
                {{ humanizeDate(changeHistory.updateTime) }}
              </span>
            </div>
          </NTooltip>
          <div v-else>
            {{ "Previously latest schema" }}
          </div>
        </div>
      </div>
    </div>

    <div class="w-full h-[32rem]">
      <SchemaDesignEditor
        :key="schemaEditorKey"
        :project="project"
        :readonly="!state.isEditing"
        :branch="schemaDesign"
      />
    </div>
    <!-- Don't show delete button in view mode. -->
    <div v-if="!viewMode">
      <BBButtonConfirm
        :style="'DELETE'"
        :button-text="$t('database.delete-this-branch')"
        :require-confirm="true"
        @confirm="deleteSchemaDesign"
      />
    </div>
  </div>

  <MergeBranchPanel
    v-if="state.showDiffEditor && mergeBranchPanelContext"
    :source-branch-name="mergeBranchPanelContext.sourceBranchName"
    :target-branch-name="mergeBranchPanelContext.targetBranchName"
    @dismiss="state.showDiffEditor = false"
    @merged="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import { cloneDeep, head, isEqual, uniqueId } from "lodash-es";
import { NButton, NDivider, NInput, NTooltip, useDialog, NTag } from "naive-ui";
import { Status } from "nice-grpc-common";
import { CSSProperties, computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "@/components/SchemaEditorV1/utils";
import {
  pushNotification,
  useChangeHistoryStore,
  useDatabaseV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import { provideSQLCheckContext } from "../SQLCheck";
import { fetchBaselineMetadataOfBranch } from "../SchemaEditorV1/utils/branch";
import MergeBranchPanel from "./MergeBranchPanel.vue";
import SchemaDesignEditor from "./SchemaDesignEditor.vue";
import { generateForkedBranchName, validateBranchName } from "./utils";

interface LocalState {
  schemaDesignTitle: string;
  isEditing: boolean;
  isEditingTitle: boolean;
  showDiffEditor: boolean;
}

const props = defineProps<{
  // Should be a schema design name of main branch.
  branch: SchemaDesign;
  viewMode?: boolean;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const changeHistoryStore = useChangeHistoryStore();
const schemaDesignStore = useSchemaDesignStore();
const { schemaDesignList } = useSchemaDesignList();
const { runSQLCheck } = provideSQLCheckContext();
const dialog = useDialog();
const state = reactive<LocalState>({
  schemaDesignTitle: "",
  isEditing: false,
  isEditingTitle: false,
  showDiffEditor: false,
});
const mergeBranchPanelContext = ref<{
  sourceBranchName: string;
  targetBranchName: string;
}>();
const schemaEditorKey = ref<string>(uniqueId());

const schemaDesign = computed(() => {
  return props.branch;
});

const parentBranch = asyncComputed(async () => {
  // Show parent branch when the current branch is a personal draft and it's not the new created one.
  if (
    schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT &&
    schemaDesign.value.baselineSheetName
  ) {
    return await schemaDesignStore.fetchSchemaDesignByName(
      schemaDesign.value.baselineSheetName,
      true /* useCache */
    );
  }
  return undefined;
}, undefined);

const changeHistory = computed(() => {
  const changeHistoryName = `${baselineDatabase.value.name}/changeHistories/${schemaDesign.value.baselineChangeHistoryId}`;
  if (
    schemaDesign.value.baselineChangeHistoryId &&
    schemaDesign.value.baselineChangeHistoryId !== String(UNKNOWN_ID)
  ) {
    return changeHistoryStore.getChangeHistoryByName(changeHistoryName);
  }
  return undefined;
});

const baselineDatabase = computed(() => {
  return databaseStore.getDatabaseByName(schemaDesign.value.baselineDatabase);
});

const project = computed(() => {
  return baselineDatabase.value.projectEntity;
});

const titleInputStyle = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    minWidth: "10rem",
    "--n-color-disabled": "transparent",
    "--n-font-size": "20px",
  };
  const border = state.isEditingTitle
    ? "1px solid var(--color-control-border)"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const prepareBaselineDatabase = async () => {
  const database = await databaseStore.getOrFetchDatabaseByName(
    schemaDesign.value.baselineDatabase
  );
  if (database.uid !== String(UNKNOWN_ID)) {
    await changeHistoryStore.getOrFetchChangeHistoryListOfDatabase(
      database.name
    );
  }
};

watch(
  () => [props.branch],
  async () => {
    state.schemaDesignTitle = schemaDesign.value.title;
    await prepareBaselineDatabase();
    // Prepare the parent branch for personal draft.
    if (
      schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT &&
      schemaDesign.value.baselineSheetName
    ) {
      await schemaDesignStore.fetchSchemaDesignByName(
        schemaDesign.value.baselineSheetName,
        true /* useCache */
      );
    }
  },
  {
    immediate: true,
  }
);

const handleBranchTitleInputBlur = async () => {
  if (state.schemaDesignTitle === "") {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Branch name cannot be empty.",
    });
    return;
  }
  if (!validateBranchName(state.schemaDesignTitle)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const updateMask = [];
  if (schemaDesign.value.title !== state.schemaDesignTitle) {
    updateMask.push("title");
  }
  if (updateMask.length !== 0) {
    await schemaDesignStore.updateSchemaDesign(
      SchemaDesign.fromPartial({
        name: schemaDesign.value.name,
        title: state.schemaDesignTitle,
        baselineDatabase: schemaDesign.value.baselineDatabase,
      }),
      updateMask
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isEditingTitle = false;
};

const handleMergeBranch = () => {
  const tempList = schemaDesignList.value.filter((item) => {
    const [projectName] = getProjectAndSchemaDesignSheetId(item.name);
    return (
      `${projectNamePrefix}${projectName}` === project.value.name &&
      item.engine === schemaDesign.value.engine &&
      item.name !== schemaDesign.value.name
    );
  });
  const targetBranchName = parentBranch.value
    ? parentBranch.value.name
    : head(tempList)?.name;
  if (!targetBranchName) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "No branch to merge.",
    });
    return;
  }

  mergeBranchPanelContext.value = {
    sourceBranchName: schemaDesign.value.name,
    targetBranchName: targetBranchName,
  };
  state.showDiffEditor = true;
};

const handleEdit = async () => {
  state.isEditing = true;
};

const handleCancelEdit = async () => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    schemaDesign.value.name
  );
  if (!branchSchema) {
    return;
  }

  const baselineMetadata = await fetchBaselineMetadataOfBranch(
    branchSchema.branch
  );
  const mergedMetadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
  );
  if (!isEqual(mergedMetadata, schemaDesign.value.schemaMetadata)) {
    // If the metadata is changed, we need to rebuild the editing state.
    schemaEditorKey.value = uniqueId();
  }
  state.isEditing = false;
};

const handleSaveBranch = async () => {
  if (!state.isEditing) {
    return;
  }
  const check = runSQLCheck.value;
  if (check && !(await check())) {
    return;
  }

  const updateMask = [];
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    schemaDesign.value.name
  );
  if (!branchSchema) {
    return;
  }

  const baselineMetadata = await fetchBaselineMetadataOfBranch(
    branchSchema.branch
  );
  const mergedMetadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
  );
  const validationMessages = validateDatabaseMetadata(mergedMetadata);
  if (validationMessages.length > 0) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Invalid schema design",
      description: validationMessages.join("\n"),
    });
    return;
  }
  if (!isEqual(mergedMetadata, schemaDesign.value.schemaMetadata)) {
    updateMask.push("metadata");
  }

  if (updateMask.length !== 0) {
    if (schemaDesign.value.type === SchemaDesign_Type.MAIN_BRANCH) {
      const branchName = generateForkedBranchName(schemaDesign.value);
      const newBranch = await schemaDesignStore.createSchemaDesignDraft({
        ...schemaDesign.value,
        baselineSchema: schemaDesign.value.schema,
        schemaMetadata: mergedMetadata,
        baselineSchemaMetadata: baselineMetadata,
        title: branchName,
      });
      try {
        await schemaDesignStore.mergeSchemaDesign({
          name: newBranch.name,
          targetName: schemaDesign.value.name,
        });
      } catch (error: any) {
        // If there is conflict, we need to show the conflict and let user resolve it.
        if (error.code === Status.FAILED_PRECONDITION) {
          dialog.create({
            negativeText: t("schema-designer.save-draft"),
            positiveText: t("schema-designer.diff-editor.resolve"),
            title: t("schema-designer.diff-editor.auto-merge-failed"),
            content: t("schema-designer.diff-editor.need-to-resolve-conflicts"),
            autoFocus: true,
            closable: true,
            maskClosable: true,
            closeOnEsc: true,
            onNegativeClick: () => {
              // Go to draft branch detail page after merge failed.
              const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
                newBranch.name
              );
              state.isEditing = false;
              router.replace({
                name: "workspace.branch.detail",
                params: {
                  projectName,
                  branchName: sheetId,
                },
              });
            },
            onPositiveClick: () => {
              state.showDiffEditor = true;
              mergeBranchPanelContext.value = {
                sourceBranchName: newBranch.name,
                targetBranchName: schemaDesign.value.name,
              };
            },
          });
        } else {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: `Request error occurred`,
            description: error.details,
          });
        }
        return;
      }

      // Delete the draft after merged.
      await schemaDesignStore.deleteSchemaDesign(newBranch.name);
      // Fetch the latest schema design after merged.
      await schemaDesignStore.fetchSchemaDesignByName(
        schemaDesign.value.name,
        false /* !useCache */
      );
    } else {
      await schemaDesignStore.updateSchemaDesign(
        SchemaDesign.fromPartial({
          name: schemaDesign.value.name,
          title: state.schemaDesignTitle,
          engine: schemaDesign.value.engine,
          baselineDatabase: schemaDesign.value.baselineDatabase,
          baselineSchema: schemaDesign.value.baselineSchema,
          schemaMetadata: mergedMetadata,
        }),
        updateMask
      );
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isEditing = false;
};

const handleMergeAfterConflictResolved = (branchName: string) => {
  state.showDiffEditor = false;
  state.isEditing = false;
  const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(branchName);
  router.replace({
    name: "workspace.branch.detail",
    params: {
      projectName,
      branchName: sheetId,
    },
  });
};

const handleApplySchemaDesignClick = () => {
  router.push({
    name: "workspace.sync-schema",
    query: {
      schemaDesignName: schemaDesign.value.name,
    },
  });
};

const deleteSchemaDesign = async () => {
  await schemaDesignStore.deleteSchemaDesign(schemaDesign.value.name);
  router.replace({
    name: "workspace.branch.dashboard",
  });
};
</script>
