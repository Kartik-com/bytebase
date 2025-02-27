<template>
  <div class="space-y-3 pt-2 w-full overflow-x-auto">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="flex flex-row justify-start items-center"></div>
      <div class="flex flex-row justify-end items-center gap-x-2">
        <NInput
          v-model:value="state.searchKeyword"
          class="!w-36"
          clearable
          :placeholder="$t('common.filter-by-name')"
        />
        <NButton type="primary" @click="handleCreateBranch">
          <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
          <span>{{ $t("database.new-branch") }}</span>
        </NButton>
      </div>
    </div>

    <BranchTable
      :branches="filteredBranches"
      :hide-project-column="true"
      :ready="ready"
      @click="handleBranchClick"
    />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import BranchTable from "@/components/Branch/BranchTable.vue";
import { useProjectV1Store } from "@/store";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";
import {
  getProjectAndSchemaDesignSheetId,
  getProjectName,
} from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

const props = defineProps<{
  projectId: string;
}>();

interface LocalState {
  searchKeyword: string;
}

const router = useRouter();
const projectV1Store = useProjectV1Store();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  searchKeyword: "",
});

const project = computed(() => projectV1Store.getProjectByUID(props.projectId));

const filteredBranches = computed(() => {
  return orderBy(
    props.projectId
      ? schemaDesignList.value.filter((schemaDesign) =>
          schemaDesign.name.startsWith(project.value.name)
        )
      : schemaDesignList.value,
    "updateTime",
    "desc"
  ).filter((branch) => {
    return state.searchKeyword
      ? branch.title.includes(state.searchKeyword)
      : true;
  });
});

const handleCreateBranch = () => {
  const projectName = getProjectName(project.value.name);
  router.push({
    name: "workspace.branch.detail",
    params: {
      projectName: projectName,
      branchName: "new",
    },
  });
};

const handleBranchClick = async (schemaDesign: SchemaDesign) => {
  const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
    schemaDesign.name
  );
  router.push({
    name: "workspace.branch.detail",
    params: {
      projectName: projectName,
      branchName: `${sheetId}`,
    },
  });
};
</script>
