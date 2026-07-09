<script setup lang="ts">

import { computed, onMounted, reactive, ref } from 'vue'

import { storeToRefs } from 'pinia'

import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'

import { fetchPlayTree, fetchPlayTemplates } from '@/api/playCatalog'

import { patchLotteryCatalog } from '@/api/lotteryCatalog'

/** P1 只读；P5 维护编辑需设 VITE_LOTTERY_CATALOG_P5=true */
const catalogMaintenanceEditable = import.meta.env.VITE_LOTTERY_CATALOG_P5 === 'true'

import type { PlayTemplateRow, PlayTypeNode, LotteryCatalogRow } from '@/types/lottery'

import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'



const catalog = useLotteryCatalogStore()

const { rows } = storeToRefs(catalog)

const onSaleCount = computed(() => rows.value.filter((r) => r.saleStatus === 'on_sale').length)
const maintenanceCount = computed(() => rows.value.filter((r) => r.saleStatus !== 'on_sale').length)

const templates = ref<PlayTemplateRow[]>([])

const selectedTemplate = ref('')

const playTypes = ref<PlayTypeNode[]>([])

const playLoading = ref(false)



const editVisible = ref(false)

const saving = ref(false)

const editing = ref<LotteryCatalogRow | null>(null)

const editForm = reactive({

  displayName: '',

  outboundLotteryCode: '',

  sortOrder: 1,

  saleStatus: 'maintenance' as 'on_sale' | 'maintenance',

})



onMounted(async () => {

  await catalog.hydrate()

  templates.value = await fetchPlayTemplates()

  if (templates.value.length > 0) {

    selectedTemplate.value = templates.value[0].code

    await loadPlayTree()

  }

})



async function loadPlayTree() {

  if (!selectedTemplate.value) return

  playLoading.value = true

  try {

    const tree = await fetchPlayTree(selectedTemplate.value)

    playTypes.value = tree.playTypes ?? []

  } finally {

    playLoading.value = false

  }

}



function saleLabel(status: string) {

  return status === 'on_sale' ? '上架' : '维护'

}



function openMaintenanceEdit(row: LotteryCatalogRow) {

  editing.value = row

  editForm.displayName = row.displayName

  editForm.outboundLotteryCode = row.outboundLotteryCode || row.code

  editForm.sortOrder = row.sortOrder

  editForm.saleStatus = row.saleStatus === 'on_sale' ? 'on_sale' : 'maintenance'

  editVisible.value = true

}



async function enterMaintenance(row: LotteryCatalogRow) {

  saving.value = true

  try {

    await patchLotteryCatalog(row.code, { enterMaintenance: true, saleStatus: 'maintenance' })

    await catalog.hydrate()

    ElMessage.success(`「${row.displayName}」已设为维护`)

  } catch (e) {

    ElMessage.error(e instanceof Error ? e.message : '操作失败')

  } finally {

    saving.value = false

  }

}



async function saveMaintenanceEdit() {

  if (!editing.value) return

  saving.value = true

  try {

    await patchLotteryCatalog(editing.value.code, {

      displayName: editForm.displayName.trim(),

      outboundLotteryCode: editForm.outboundLotteryCode.trim(),

      sortOrder: editForm.sortOrder,

      saleStatus: editForm.saleStatus,

    })

    await catalog.hydrate()

    editVisible.value = false

    ElMessage.success('彩种维护信息已保存')

  } catch (e) {

    ElMessage.error(e instanceof Error ? e.message : '保存失败')

  } finally {

    saving.value = false

  }

}

</script>



<template>

  <div>

    <h1 class="admin-page-title">彩种目录</h1>

    <p style="margin: 0 0 1rem; font-size: 13px; color: var(--el-text-color-secondary)">
      当前共 {{ rows.length }} 个彩种：上架 <strong>{{ onSaleCount }}</strong> 个，维护 <strong>{{ maintenanceCount }}</strong> 个。
    </p>



    <el-table :data="rows" stripe style="width: 100%; margin-bottom: 2rem">

      <el-table-column prop="sortOrder" label="排序" min-width="64" />

      <el-table-column prop="code" label="code" min-width="128" />

      <el-table-column prop="displayName" label="对外中文名" min-width="140" />

      <el-table-column prop="playTemplate" label="玩法模板" min-width="100" />

      <el-table-column prop="categoryCode" label="大类" min-width="72" />

      <el-table-column prop="outboundLotteryCode" label="对接码" min-width="128" />

      <el-table-column label="状态" min-width="88">

        <template #default="{ row }">

          <el-tag :type="row.saleStatus === 'on_sale' ? 'success' : 'warning'" size="small">

            {{ saleLabel(row.saleStatus) }}

          </el-tag>

        </template>

      </el-table-column>

      <el-table-column label="操作" v-if="catalogMaintenanceEditable" min-width="160" fixed="right">

        <template #default="{ row }">

          <el-button

            v-if="row.saleStatus === 'on_sale'"

            link

            type="warning"

            :loading="saving"

            @click="enterMaintenance(row)"

          >

            设为维护

          </el-button>

          <el-button

            v-else

            link

            type="primary"

            @click="openMaintenanceEdit(row)"

          >

            维护编辑

          </el-button>

        </template>

      </el-table-column>

    </el-table>



    <AdminDialog v-if="catalogMaintenanceEditable" v-model="editVisible" title="维护态编辑" width="480px" destroy-on-close>

      <el-form label-width="108px">

        <el-form-item label="code">

          <el-input :model-value="editing?.code" disabled />

        </el-form-item>

        <el-form-item label="对外中文名">

          <el-input v-model="editForm.displayName" maxlength="32" show-word-limit />

        </el-form-item>

        <el-form-item label="对接码">

          <el-input v-model="editForm.outboundLotteryCode" maxlength="64" />

        </el-form-item>

        <el-form-item label="排序">

          <el-input-number v-model="editForm.sortOrder" :min="1" :max="999" />

        </el-form-item>

        <el-form-item label="状态">

          <el-radio-group v-model="editForm.saleStatus">

            <el-radio-button value="maintenance">维护</el-radio-button>

            <el-radio-button value="on_sale">上架</el-radio-button>

          </el-radio-group>

        </el-form-item>

      </el-form>

      <template #footer>

        <el-button @click="editVisible = false">取消</el-button>

        <el-button type="primary" :loading="saving" @click="saveMaintenanceEdit">保存</el-button>

      </template>

    </AdminDialog>



    <h2 style="font-size: 1.125rem; margin: 0 0 0.75rem">玩法目录（只读）</h2>

    <div style="margin-bottom: 1rem; display: flex; gap: 0.75rem; align-items: center">

      <span style="font-size: 13px; color: var(--el-text-color-secondary)">玩法模板</span>

      <el-select

        v-model="selectedTemplate"

        style="width: 220px"

        @change="loadPlayTree"

      >

        <el-option

          v-for="tpl in templates"

          :key="tpl.code"

          :label="`${tpl.label} (${tpl.code})`"

          :value="tpl.code"

        />

      </el-select>

    </div>



    <el-table v-loading="playLoading" :data="playTypes" stripe row-key="typeId">

      <el-table-column type="expand">

        <template #default="{ row }">

          <el-table :data="row.subPlays" size="small" style="width: 100%">

            <el-table-column prop="subId" label="sub_id" min-width="160" />

            <el-table-column prop="label" label="子玩法" min-width="160" />

            <el-table-column prop="outboundPlayCode" label="对外玩法码" min-width="240" />

            <el-table-column prop="betMode" label="bet_mode" min-width="100" />

          </el-table>

        </template>

      </el-table-column>

      <el-table-column prop="typeId" label="type_id" min-width="120" />

      <el-table-column prop="label" label="玩法类型" min-width="120" />

      <el-table-column label="子玩法数" min-width="88">

        <template #default="{ row }">{{ row.subPlays?.length ?? 0 }}</template>

      </el-table-column>

    </el-table>

  </div>

</template>


