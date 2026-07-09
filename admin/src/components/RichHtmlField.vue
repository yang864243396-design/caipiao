<script setup lang="ts">
import '@wangeditor/editor/dist/css/style.css'
import { onBeforeUnmount, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor/editor'
import { uploadCmsImage } from '@/api/upload'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const editorRef = shallowRef<IDomEditor>()
const innerHtml = shallowRef(normalizeHtml(props.modelValue))

watch(
  () => props.modelValue,
  (v) => {
    const next = normalizeHtml(v)
    if (next === innerHtml.value) return
    innerHtml.value = next
    const editor = editorRef.value
    if (editor && editor.getHtml() !== next) {
      editor.setHtml(next)
    }
  },
)

const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: ['group-video', 'fullScreen'],
}

const editorConfig: Partial<IEditorConfig> = {
  placeholder: '请输入正文内容…',
  MENU_CONF: {
    uploadImage: {
      async customUpload(
        file: File,
        insertFn: (url: string, alt: string, href: string) => void,
      ) {
        try {
          const url = await uploadCmsImage(file)
          insertFn(url, file.name, url)
        } catch (e) {
          ElMessage.error(e instanceof Error ? e.message : '图片上传失败')
        }
      },
    },
  },
}

function normalizeHtml(raw: string): string {
  const t = raw?.trim() ?? ''
  return t || '<p><br></p>'
}

function isEmptyHtml(html: string): boolean {
  const t = html.replace(/\s/g, '')
  return t === '' || t === '<p><br></p>' || t === '<p></p>'
}

function handleCreated(editor: IDomEditor) {
  editorRef.value = editor
}

function handleChange(editor: IDomEditor) {
  const html = editor.getHtml()
  innerHtml.value = html
  emit('update:modelValue', isEmptyHtml(html) ? '' : html)
}

onBeforeUnmount(() => {
  const editor = editorRef.value
  if (editor == null) return
  editor.destroy()
})
</script>

<template>
  <div class="rich-html-field">
    <div class="rich-html-field__editor">
      <Toolbar
        class="rich-html-field__toolbar"
        :editor="editorRef"
        :default-config="toolbarConfig"
        mode="default"
      />
      <Editor
        v-model="innerHtml"
        class="rich-html-field__body"
        :default-config="editorConfig"
        mode="default"
        @on-created="handleCreated"
        @on-change="handleChange"
      />
    </div>
  </div>
</template>

<style scoped>
.rich-html-field {
  width: 100%;
}

.rich-html-field__editor {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.rich-html-field__toolbar {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.rich-html-field__body {
  height: 320px !important;
  overflow-y: hidden;
}

:deep(.w-e-text-container) {
  background: var(--el-bg-color);
}

:deep(.w-e-scroll) {
  min-height: 280px !important;
}
</style>
