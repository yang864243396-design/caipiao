<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage, type TreeInstance } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import {
  ADMIN_MENU_ALL_NODE_ID,
  ADMIN_MENU_ROUTE_TREE,
  type AdminMenuRouteNode,
} from '@/constants/adminMenuRouteTree'
import {
  checkedKeysFromMenuPaths,
  formatMenuLabelsSummary,
  menuPathsFromCheckedNodes,
} from '@/utils/adminMenuRouteTree'
import { useAdminRolesStore, type AdminRole } from '@/stores/adminRoles'

const store = useAdminRolesStore()
const { roles, loading } = storeToRefs(store)

onMounted(() => {
  void store.hydrate()
})

const dialog = ref(false)
const editing = ref<AdminRole | null>(null)
const form = reactive<AdminRole>({ id: '', name: '', menuPaths: [] })
const treeRef = ref<TreeInstance>()

function syncTreeChecked(menuPaths: string[]) {
  void nextTick(() => {
    treeRef.value?.setCheckedKeys(checkedKeysFromMenuPaths(menuPaths), false)
  })
}

function openNew() {
  editing.value = null
  form.id = `r_${Date.now().toString(36)}`
  form.name = '新角色'
  form.menuPaths = ['/dashboard']
  dialog.value = true
  syncTreeChecked(form.menuPaths)
}

function openEdit(row: AdminRole) {
  editing.value = row
  form.id = row.id
  form.name = row.name
  form.menuPaths = [...row.menuPaths]
  dialog.value = true
  syncTreeChecked(form.menuPaths)
}

function onTreeCheck(data: AdminMenuRouteNode, ctx: { checkedKeys: (string | number)[] }) {
  const tree = treeRef.value
  if (!tree) return

  if (data.id === ADMIN_MENU_ALL_NODE_ID && ctx.checkedKeys.includes(ADMIN_MENU_ALL_NODE_ID)) {
    tree.setCheckedKeys([ADMIN_MENU_ALL_NODE_ID], false)
    return
  }

  if (data.id !== ADMIN_MENU_ALL_NODE_ID && ctx.checkedKeys.includes(ADMIN_MENU_ALL_NODE_ID)) {
    tree.setChecked(ADMIN_MENU_ALL_NODE_ID, false, false)
  }
}

async function saveForm() {
  const tree = treeRef.value
  if (!tree) return

  const checkedNodes = tree.getCheckedNodes(false, false) as AdminMenuRouteNode[]
  const menuPaths = menuPathsFromCheckedNodes(checkedNodes)
  if (!menuPaths.length) {
    ElMessage.warning('请至少勾选一个菜单权限')
    return
  }

  form.menuPaths = menuPaths
  try {
    await store.upsertRole({
      id: form.id,
      name: form.name.trim() || '未命名',
      menuPaths: [...form.menuPaths],
    })
    ElMessage.success('已保存')
    dialog.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function remove(row: AdminRole) {
  const ok = await adminConfirmDialog({
    title: '确认',
    message: `删除角色「${row.name}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    const deleted = await store.removeRole(row.id)
    ElMessage[deleted ? 'success' : 'warning'](deleted ? '已删除' : '不可删除内置超级管理员')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>

<template>
  <div>
    <h1 class="admin-page-title">角色管理</h1>

    <div class="toolbar">
      <el-button type="primary" @click="openNew">新增角色</el-button>
    </div>

    <el-table v-loading="loading" :data="roles" stripe style="width: 100%">
      <el-table-column prop="id" label="角色 id" min-width="120" />
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="可见菜单" min-width="280">
        <template #default="{ row }">
          <span class="paths-summary">{{ formatMenuLabelsSummary(row.menuPaths) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" :title="editing ? '编辑角色' : '新增角色'" width="min(92vw, 560px)">
      <el-form label-position="top">
        <el-form-item label="角色 id">
          <el-input v-model="form.id" :disabled="!!editing" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="可见菜单">
          <div class="route-tree-wrap">
            <el-tree ref="treeRef" :data="ADMIN_MENU_ROUTE_TREE" show-checkbox node-key="id" default-expand-all
              :props="{ label: 'label', children: 'children' }" @check="onTreeCheck" />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="saveForm">保存</el-button>
      </template>
    </AdminDialog>
  </div>
</template>

<style scoped>
.admin-page-desc {
  margin: 0 0 1rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.toolbar {
  margin-bottom: 1rem;
}

.paths-summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.route-tree-wrap {
  width: 100%;
  max-height: 360px;
  overflow: auto;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}
</style>
