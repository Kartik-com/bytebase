<template>
  <div class="flex flex-col items-stretch gap-y-4 overflow-x-hidden">
    <div class="flex flex-col gap-y-2 max-w-max">
      <div
        v-if="changes.length === 0"
        class="text-control-placeholder text-sm leading-[28px]"
      >
        {{
          $t(
            "changelist.add-change.change-history.select-at-least-one-history-below"
          )
        }}
      </div>
      <ChangeHistoryChangeItem
        v-for="change in changes"
        :key="change.source"
        :change="change"
        @click-item="handleClickChange($event)"
        @remove-item="handleRemoveChange($event)"
      />
    </div>
    <div class="flex flex-row items-center justify-between py-0.5">
      <div class="flex flex-row items-center justify-start gap-x-2">
        <DatabaseSelect
          v-model:database="state.databaseUID"
          :project="project.uid"
        />
        <NSelect
          v-model:value="state.affectedTableKey"
          :options="affectedTableOptions"
          style="width: 12rem"
        />
        <NCheckboxGroup v-model:value="state.changeHistoryTypes">
          <NCheckbox :value="ChangeHistory_Type.MIGRATE">DDL</NCheckbox>
          <NCheckbox :value="ChangeHistory_Type.DATA">DML</NCheckbox>
        </NCheckboxGroup>
      </div>
      <div class="flex flex-row items-center justify-end gap-x-2">
        <SearchBox
          v-model:value="state.keyword"
          :placeholder="$t('common.filter-by-name')"
        />
      </div>
    </div>
    <ChangeHistoryTable
      v-model:selected="selectedChangeHistoryList"
      :change-history-list="filteredChangeHistoryList"
      :is-fetching="state.isLoading"
      :keyword="state.keyword"
      class="w-full"
      @click-item="state.detailChangeHistoryName = $event.name"
    />

    <ChangeHistoryDetailPanel
      :change-history-name="state.detailChangeHistoryName"
      @close="state.detailChangeHistoryName = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import { first } from "lodash-es";
import { NCheckbox, NCheckboxGroup, NSelect, SelectOption } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { h } from "vue";
import { DatabaseSelect, SearchBox } from "@/components/v2";
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useLocalSheetStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { AffectedTable, EmptyAffectedTable } from "@/types/changeHistory";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import {
  ChangeHistory,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import {
  extractDatabaseResourceName,
  getAffectedTablesOfChangeHistory,
  keyBy,
  setSheetStatement,
} from "@/utils";
import ChangeHistoryDetailPanel from "../../ChangeHistoryDetailPanel";
import { useChangelistDetailContext } from "../../context";
import { useAddChangeContext } from "../context";
import ChangeHistoryChangeItem from "./ChangeHistoryChangeItem.vue";
import ChangeHistoryTable from "./ChangeHistoryTable";
import {
  getAffectedTableDisplayName,
  getAffectedTableKey,
  getAffectedTablesFromChangeHistoryList,
  semanticChangeHistoryType,
} from "./utils";

type LocalState = {
  isLoading: boolean;
  keyword: string;
  databaseUID: string | undefined;
  changeHistoryList: ChangeHistory[];
  affectedTableKey: string | undefined;
  changeHistoryTypes: ChangeHistory_Type[];
  detailChangeHistoryName: string | undefined;
};

type AffectedTableSelectOption = SelectOption & {
  affectedTable: AffectedTable;
  value: string;
};

const ALL_TABLE_KEY = getAffectedTableKey(EmptyAffectedTable);

const { project } = useChangelistDetailContext();
const { changesFromChangeHistory: changes } = useAddChangeContext();
const localSheetStore = useLocalSheetStore();

const state = reactive<LocalState>({
  isLoading: false,
  keyword: "",
  databaseUID: undefined,
  changeHistoryList: [],
  affectedTableKey: undefined,
  changeHistoryTypes: [ChangeHistory_Type.DATA, ChangeHistory_Type.MIGRATE],
  detailChangeHistoryName: undefined,
});

const database = computed(() => {
  const uid = state.databaseUID;
  if (!uid || uid === String(UNKNOWN_ID)) return undefined;
  return useDatabaseV1Store().getDatabaseByUID(uid);
});

const affectedTableOptions = computed(() => {
  const affectedTables = getAffectedTablesFromChangeHistoryList(
    state.changeHistoryList
  );

  return affectedTables.map<AffectedTableSelectOption>((item) => {
    const key = getAffectedTableKey(item);
    const name = getAffectedTableDisplayName(item);
    return {
      label: name,
      value: key,
      affectedTable: item,
      renderLabel() {
        const classes = ["truncate"];
        if (item.dropped) {
          classes.push("text-gray-400");
        }
        return h("span", { class: classes, "data-key": key }, name);
      },
    };
  });
});

const filteredChangeHistoryList = computed(() => {
  let list = [...state.changeHistoryList];
  const types = state.changeHistoryTypes;
  list = list.filter((changeHistory) => {
    const semanticType = semanticChangeHistoryType(changeHistory.type);
    return types.includes(semanticType);
  });

  const kw = state.keyword.trim().toLowerCase();
  if (kw) {
    list = list.filter((changeHistory) =>
      changeHistory.version.toLowerCase().includes(kw)
    );
  }
  const tableKey = state.affectedTableKey;
  if (tableKey && tableKey !== ALL_TABLE_KEY) {
    list = list.filter((changeHistory) => {
      const affectedTables = getAffectedTablesOfChangeHistory(changeHistory);
      return affectedTables.find(
        (item) => getAffectedTableKey(item) === tableKey
      );
    });
  }

  return list;
});

const selectedChangeHistoryList = computed<string[]>({
  get() {
    return changes.value.map((change) => {
      return change.source;
    });
  },
  set(selected) {
    const existedChangesByChangeHistoryName = keyBy(
      changes.value,
      (change) => change.source
    );
    const updatedChanges: Change[] = [];
    for (let i = 0; i < selected.length; i++) {
      const name = selected[i];
      const changeHistory =
        useChangeHistoryStore().getChangeHistoryByName(name);
      if (changeHistory) {
        const existedChange = existedChangesByChangeHistoryName.get(name);
        if (existedChange) {
          updatedChanges.push(existedChange);
        } else {
          const uid = localSheetStore.nextUID();
          const sheet = localSheetStore.createLocalSheet(
            `${project.value.name}/sheets/${uid}`
          );
          setSheetStatement(sheet, changeHistory.statement);
          updatedChanges.push({
            sheet: sheet.name,
            source: changeHistory.name,
          });
        }
      }
    }
    changes.value = updatedChanges;
  },
});

const fetchChangeHistoryList = async () => {
  const db = database.value;
  if (!db) {
    state.changeHistoryList = [];
    return;
  }

  state.isLoading = true;
  const name = db.name;
  await useDBSchemaV1Store().getOrFetchDatabaseMetadata(
    name,
    false /* !skipCache */,
    true /* silent */
  );
  const changeHistoryList =
    await useChangeHistoryStore().getOrFetchChangeHistoryListOfDatabase(name);
  // Check if the state is still valid
  if (name === database.value?.name) {
    state.changeHistoryList = changeHistoryList;
  }
  state.isLoading = false;
};

const handleRemoveChange = (change: Change) => {
  const index = changes.value.findIndex((c) => c.source === change.source);
  if (index >= 0) {
    changes.value.splice(index, 1);
  }
};

const handleClickChange = (change: Change) => {
  const changeHistoryName = change.source;
  const database = useDatabaseV1Store().getDatabaseByName(
    extractDatabaseResourceName(changeHistoryName).full
  );
  state.databaseUID = database.uid;
  state.detailChangeHistoryName = changeHistoryName;
};

// Select the first database automatically
watch(
  () => project.value.name,
  (project) => {
    const databaseList = useDatabaseV1Store().databaseListByProject(project);
    state.databaseUID = first(databaseList)?.uid;
  },
  { immediate: true }
);

watch(() => database.value?.uid, fetchChangeHistoryList, { immediate: true });

watch(
  affectedTableOptions,
  (options) => {
    state.affectedTableKey = first(options)?.value;
  },
  { immediate: true }
);
</script>
