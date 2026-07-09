<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: string | number
    destroyOnClose?: boolean
    alignCenter?: boolean
    appendToBody?: boolean
  }>(),
  {
    destroyOnClose: true,
    alignCenter: true,
    appendToBody: true,
    width: '520px',
  },
)

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  closed: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="width"
    :align-center="alignCenter"
    :append-to-body="appendToBody"
    :destroy-on-close="destroyOnClose"
    class="admin-dialog"
    @closed="emit('closed')"
  >
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>
