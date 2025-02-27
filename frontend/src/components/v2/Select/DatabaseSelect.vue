<template>
  <NSelect
    :value="database"
    :options="options"
    :placeholder="placeholder ?? $t('database.select')"
    :virtual-scroll="true"
    :filter="filterByDatabaseName"
    :filterable="true"
    class="bb-database-select"
    style="width: 12rem"
    v-bind="$attrs"
    :render-label="renderLabel"
    @update:value="$emit('update:database', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption, SelectRenderLabel } from "naive-ui";
import { computed, h, watch } from "vue";
import { useSlots } from "vue";
import { useI18n } from "vue-i18n";
import {
  useCurrentUserV1,
  useSearchDatabaseV1List,
  useDatabaseV1Store,
} from "@/store";
import { ComposedDatabase, UNKNOWN_ID, unknownDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { instanceV1Name, supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model";

interface DatabaseSelectOption extends SelectOption {
  value: string;
  database: ComposedDatabase;
}

const slots = useSlots();
const props = withDefaults(
  defineProps<{
    database?: string;
    environment?: string;
    instance?: string;
    project?: string;
    allowedEngineTypeList?: readonly Engine[];
    includeAll?: boolean;
    autoReset?: boolean;
    placeholder?: string;
    filter?: (database: ComposedDatabase, index: number) => boolean;
  }>(),
  {
    database: undefined,
    environment: undefined,
    instance: undefined,
    project: undefined,
    allowedEngineTypeList: () => supportedEngineV1List(),
    includeAll: false,
    autoReset: true,
    placeholder: undefined,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:database", value: string | undefined): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const { ready } = useSearchDatabaseV1List({
  parent: "instances/-",
});

const rawDatabaseList = computed(() => {
  const list = useDatabaseV1Store().databaseListByUser(currentUserV1.value);

  return list.filter((db) => {
    if (props.environment && props.environment !== String(UNKNOWN_ID)) {
      if (db.effectiveEnvironmentEntity.uid !== props.environment) {
        return false;
      }
    }
    if (props.instance && props.instance !== String(UNKNOWN_ID)) {
      if (db.instanceEntity.uid !== props.instance) return false;
    }
    if (props.project && props.project !== String(UNKNOWN_ID)) {
      if (db.projectEntity.uid !== props.project) return false;
    }
    if (!props.allowedEngineTypeList.includes(db.instanceEntity.engine)) {
      return false;
    }

    return true;
  });
});

const combinedDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.includeAll) {
    const dummyAll = {
      ...unknownDatabase(),
      databaseName: t("database.all"),
    };
    list.unshift(dummyAll);
  }

  return list;
});

const options = computed(() => {
  return combinedDatabaseList.value.map<DatabaseSelectOption>((database) => {
    return {
      database,
      value: database.uid,
      label: database.databaseName,
    };
  });
});

const renderLabel: SelectRenderLabel = (option) => {
  const { database } = option as DatabaseSelectOption;
  if (!database) {
    return;
  }

  if (slots.default) {
    return slots.default({ database });
  }

  const children = [h("div", {}, [database.databaseName])];
  if (database.uid !== String(UNKNOWN_ID)) {
    // prefix engine icon
    children.unshift(
      h(InstanceV1EngineIcon, {
        class: "mr-1",
        instance: database.instanceEntity,
      })
    );
    // suffix engine name
    children.push(
      h(
        "div",
        {
          class: "text-xs opacity-60 ml-1",
        },
        [`(${instanceV1Name(database.instanceEntity)})`]
      )
    );
  }
  return h(
    "div",
    {
      class: "w-full flex flex-row justify-start items-center truncate",
    },
    children
  );
};

const filterByDatabaseName = (pattern: string, option: SelectOption) => {
  const { database } = option as DatabaseSelectOption;
  return database.databaseName.toLowerCase().includes(pattern.toLowerCase());
};

// The database list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    ready.value &&
    props.database &&
    !combinedDatabaseList.value.find((item) => item.uid === props.database)
  ) {
    emit("update:database", undefined);
  }
};

watch(
  [
    () => [props.project, props.environment, props.database],
    combinedDatabaseList,
  ],
  resetInvalidSelection,
  {
    immediate: true,
  }
);
</script>

<style lang="postcss" scoped>
.bb-database-select :deep(.n-base-selection-input:focus) {
  @apply !ring-0;
}
</style>
