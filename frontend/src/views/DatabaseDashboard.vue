<template>
  <div class="flex flex-col relative">
    <div class="px-4 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :include-all="true"
        :environment="selectedEnvironment?.name"
        @update:environment="changeEnvironment"
      />

      <div class="flex items-center space-x-4">
        <NTooltip v-if="canVisitUnassignedDatabases">
          <template #trigger>
            <router-link
              :to="{
                name: 'workspace.project.detail',
                params: {
                  projectSlug: DEFAULT_PROJECT_ID,
                },
                hash: '#databases',
              }"
              class="normal-link text-sm"
            >
              {{ $t("database.view-unassigned-databases") }}
            </router-link>
          </template>

          <div class="whitespace-pre-wrap">
            {{ $t("quick-action.unassigned-db-hint") }}
          </div>
        </NTooltip>

        <NInputGroup style="width: auto">
          <InstanceSelect
            class="!w-48"
            :instance="state.instanceFilter"
            :include-all="true"
            :environment="selectedEnvironment?.uid"
            @update:instance="
              state.instanceFilter = $event ?? String(UNKNOWN_ID)
            "
          />
          <SearchBox
            :value="state.searchText"
            :placeholder="$t('database.filter-database')"
            :autofocus="true"
            @update:value="changeSearchText($event)"
          />
        </NInputGroup>
      </div>
    </div>

    <DatabaseOperations
      v-if="selectedDatabases.length > 0"
      class="mb-3"
      :databases="selectedDatabases"
      @dismiss="state.selectedDatabaseIds.clear()"
    />

    <DatabaseV1Table
      pagination-class="mb-4"
      :database-list="filteredDatabaseList"
      :database-group-list="filteredDatabaseGroupList"
      :show-placeholder="true"
      :show-selection-column="true"
    >
      <template #selection-all="{ databaseList }">
        <input
          v-if="databaseList.length > 0"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          v-bind="getAllSelectionState(databaseList)"
          @input="
            toggleDatabasesSelection(
              databaseList,
              ($event.target as HTMLInputElement).checked
            )
          "
        />
      </template>
      <template #selection="{ database }">
        <input
          v-if="isDatabase(database)"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="isDatabaseSelected(database as ComposedDatabase)"
          @click.stop="
            toggleDatabasesSelection(
              [database],
              ($event.target as HTMLInputElement).checked
            )
          "
        />
        <div v-else class="text-control-light cursor-not-allowed ml-auto">
          -
        </div>
      </template>
    </DatabaseV1Table>

    <div
      v-if="state.loading"
      class="absolute inset-0 bg-white/50 flex justify-center items-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInputGroup, NTooltip } from "naive-ui";
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  EnvironmentTabFilter,
  InstanceSelect,
  DatabaseV1Table,
  SearchBox,
} from "@/components/v2";
import { isDatabase } from "@/components/v2/Model/DatabaseV1Table/utils";
import {
  useCurrentUserV1,
  useDBGroupStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  usePolicyV1Store,
  useProjectV1ListByCurrentUser,
  useUIStateStore,
} from "@/store";
import {
  Policy,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import {
  filterDatabaseV1ByKeyword,
  hasWorkspacePermissionV1,
  sortDatabaseV1List,
  isDatabaseV1Accessible,
} from "@/utils";
import {
  UNKNOWN_ID,
  UNKNOWN_ENVIRONMENT_NAME,
  DEFAULT_PROJECT_ID,
  UNKNOWN_USER_NAME,
  ComposedDatabase,
  ComposedDatabaseGroup,
  DEFAULT_PROJECT_V1_NAME,
} from "../types";

interface LocalState {
  instanceFilter: string;
  searchText: string;
  databaseGroupList: ComposedDatabaseGroup[];
  loading: boolean;
  selectedDatabaseIds: Set<string>;
}

const route = useRoute();
const router = useRouter();
const uiStateStore = useUIStateStore();
const environmentV1Store = useEnvironmentV1Store();
const { projectList } = useProjectV1ListByCurrentUser();

const state = reactive<LocalState>({
  instanceFilter: String(UNKNOWN_ID),
  searchText: "",
  databaseGroupList: [],
  loading: false,
  selectedDatabaseIds: new Set(),
});

const currentUserV1 = useCurrentUserV1();
const databaseV1Store = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const policyList = ref<Policy[]>([]);

const preparePolicyList = () => {
  usePolicyV1Store()
    .fetchPolicies({
      policyType: PolicyType.WORKSPACE_IAM,
      resourceType: PolicyResourceType.WORKSPACE,
    })
    .then((list) => (policyList.value = list));
};

watchEffect(preparePolicyList);

const selectedEnvironment = computed(() => {
  const { environment } = route.query;
  return environment
    ? environmentV1Store.getEnvironmentByName(environment as string)
    : undefined;
});

const canVisitUnassignedDatabases = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("database.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "database.visit",
      newState: true,
    });
  }
});

const prepareDatabaseList = async () => {
  // It will also be called when user logout
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    state.loading = true;
    await databaseV1Store.searchDatabaseList({
      parent: "instances/-",
    });
    state.loading = false;
  }
};

const databaseV1List = computed(() => {
  return sortDatabaseV1List(databaseV1Store.databaseList).filter((db) =>
    projectList.value.map((project) => project.name).includes(db.project)
  );
});

const prepareDatabaseGroupList = async () => {
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    state.databaseGroupList = (
      await dbGroupStore.fetchAllDatabaseGroupList()
    ).filter((dbGroup) =>
      projectList.value
        .map((project) => project.name)
        .includes(dbGroup.project.name)
    );
  }
};

watchEffect(async () => {
  state.loading = true;
  await prepareDatabaseList();
  await prepareDatabaseGroupList();
  state.loading = false;
});

const changeEnvironment = (environment: string | undefined) => {
  if (environment && environment !== UNKNOWN_ENVIRONMENT_NAME) {
    router.replace({
      name: "workspace.database",
      query: { environment },
    });
  } else {
    router.replace({ name: "workspace.database" });
  }
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

const filteredDatabaseList = computed(() => {
  let list = databaseV1List.value
    .filter((database) => database.project !== DEFAULT_PROJECT_V1_NAME)
    .filter((database) =>
      isDatabaseV1Accessible(database, currentUserV1.value)
    );
  const environment = selectedEnvironment.value;
  if (environment && environment.name !== UNKNOWN_ENVIRONMENT_NAME) {
    list = list.filter((db) => db.effectiveEnvironment === environment.name);
  }
  if (state.instanceFilter !== String(UNKNOWN_ID)) {
    list = list.filter(
      (db) => db.instanceEntity.uid === String(state.instanceFilter)
    );
  }
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
        "project",
      ])
    );
  }
  return list;
});

const filteredDatabaseGroupList = computed(() => {
  let list = [...state.databaseGroupList];
  const environment = selectedEnvironment.value;
  if (environment && environment.name !== UNKNOWN_ENVIRONMENT_NAME) {
    list = list.filter(
      (dbGroup) => dbGroup.environmentName === environment.name
    );
  }
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    list = list.filter(
      (dbGroup) =>
        dbGroup.name.includes(keyword) ||
        dbGroup.databasePlaceholder.includes(keyword)
    );
  }
  return list;
});

const getAllSelectionState = (
  databaseList: (ComposedDatabase | ComposedDatabaseGroup)[]
): { checked: boolean; indeterminate: boolean } => {
  const filteredDatabases = databaseList.filter((db) =>
    isDatabase(db)
  ) as ComposedDatabase[];

  const checked = filteredDatabases.every((db) =>
    state.selectedDatabaseIds.has(db.uid)
  );
  const indeterminate =
    !checked &&
    filteredDatabases.some((db) => state.selectedDatabaseIds.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const toggleDatabasesSelection = (
  databaseList: (ComposedDatabase | ComposedDatabaseGroup)[],
  on: boolean
): void => {
  if (on) {
    databaseList.forEach((db) => {
      if (isDatabase(db)) {
        state.selectedDatabaseIds.add((db as ComposedDatabase).uid);
      }
    });
  } else {
    databaseList.forEach((db) => {
      if (isDatabase(db)) {
        state.selectedDatabaseIds.delete((db as ComposedDatabase).uid);
      }
    });
  }
};

const isDatabaseSelected = (database: ComposedDatabase): boolean => {
  return state.selectedDatabaseIds.has((database as ComposedDatabase).uid);
};

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter(
    (db) => isDatabase(db) && state.selectedDatabaseIds.has(db.uid)
  );
});
</script>
