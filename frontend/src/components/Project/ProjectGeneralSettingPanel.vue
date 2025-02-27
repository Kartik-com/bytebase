<template>
  <form class="max-w-3xl w-full space-y-4 mx-auto">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.general") }}
    </p>
    <div class="flex justify-start items-start gap-12">
      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.name") }} <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            id="projectName"
            v-model="state.title"
            :disabled="!allowEdit"
            required
            autocomplete="off"
            type="text"
            class="textfield"
          />
        </dd>
        <div class="mt-1">
          <ResourceIdField
            resource-type="project"
            :value="extractProjectResourceName(project.name)"
            :readonly="true"
          />
        </div>
      </dl>

      <dl class="">
        <dt class="flex text-sm font-medium text-control-light">
          {{ $t("common.key") }}
          <NTooltip>
            <template #trigger>
              <heroicons-outline:information-circle class="ml-1 w-4 h-auto" />
            </template>
            {{ $t("project.key-hint") }}
          </NTooltip>
          <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            id="projectKey"
            v-model="state.key"
            :disabled="!allowEdit"
            required
            autocomplete="off"
            type="text"
            class="textfield uppercase"
          />
        </dd>
      </dl>
    </div>

    <div class="flex flex-col">
      <div for="name" class="text-sm font-medium text-control-light">
        {{ $t("common.mode") }}
        <span class="text-red-600">*</span>
      </div>
      <div class="mt-2 textlabel">
        <div class="radio-set-row">
          <label class="radio">
            <input
              v-model="state.tenantMode"
              :disabled="!allowEdit"
              tabindex="-1"
              type="radio"
              class="btn disabled:opacity-50 disabled:cursor-not-allowed"
              :value="TenantMode.TENANT_MODE_DISABLED"
            />
            <span class="label">{{ $t("project.mode.standard") }}</span>
          </label>
          <label class="radio space-x-1">
            <input
              v-model="state.tenantMode"
              :disabled="!allowEdit"
              tabindex="-1"
              type="radio"
              class="btn disabled:opacity-50 disabled:cursor-not-allowed"
              :value="TenantMode.TENANT_MODE_ENABLED"
            />
            <span class="label">{{ $t("project.mode.batch") }}</span>
            <LearnMoreLink
              url="https://www.bytebase.com/docs/concepts/batch-mode/?source=console"
            />
            <FeatureBadge feature="bb.feature.multi-tenancy" />
          </label>
        </div>
      </div>
    </div>

    <div v-if="allowEdit" class="flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="!allowSave"
        @click.prevent="save"
      >
        {{ $t("common.update") }}
      </button>
    </div>

    <FeatureModal
      :open="!!state.requiredFeature"
      :feature="state.requiredFeature"
      @cancel="state.requiredFeature = undefined"
    />
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { hasFeature, pushNotification, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_ID, FeatureType } from "@/types";
import { Project, TenantMode } from "@/types/proto/v1/project_service";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  title: string;
  key: string;
  tenantMode: TenantMode;
  requiredFeature: FeatureType | undefined;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});

const { t } = useI18n();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  title: props.project.title,
  key: props.project.key,
  tenantMode: props.project.tenantMode,
  requiredFeature: undefined,
});

const allowSave = computed((): boolean => {
  return (
    parseInt(props.project.uid, 10) !== DEFAULT_PROJECT_ID &&
    !isEmpty(state.title) &&
    (state.title !== props.project.title ||
      state.key !== props.project.key ||
      state.tenantMode !== props.project.tenantMode)
  );
});

const save = () => {
  const projectPatch = cloneDeep(props.project);
  const updateMask: string[] = [];
  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    updateMask.push("title");
  }
  if (state.key !== props.project.key) {
    projectPatch.key = state.key;
    updateMask.push("key");
  }
  if (state.tenantMode !== props.project.tenantMode) {
    if (state.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
      if (!hasFeature("bb.feature.multi-tenancy")) {
        state.tenantMode = TenantMode.TENANT_MODE_DISABLED;
        state.requiredFeature = "bb.feature.multi-tenancy";
        return;
      }
    }
    projectPatch.tenantMode = state.tenantMode;
    updateMask.push("tenant_mode");
  }
  projectV1Store.updateProject(projectPatch, updateMask).then((updated) => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-updated"),
    });
    state.title = updated.title;
    state.key = updated.key;
    state.tenantMode = updated.tenantMode;
  });
};
</script>
