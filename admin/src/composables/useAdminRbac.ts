import { computed, onMounted } from 'vue'

import { storeToRefs } from 'pinia'

import { useAuthStore } from '@/stores/auth'

import { useAdminRolesStore } from '@/stores/adminRoles'

import { canAccessPath } from '@/utils/menuRbac'



export function useAdminRbac() {

  const auth = useAuthStore()

  const rolesStore = useAdminRolesStore()

  const { roles } = storeToRefs(rolesStore)



  onMounted(() => {

    void rolesStore.hydrate().then(() => auth.syncSessionRole())

  })



  const activeRole = computed(() => {

    const id = auth.adminRoleId

    return roles.value.find((r) => r.id === id) ?? roles.value.find((r) => r.id === 'r_super')

  })



  const menuPaths = computed(() => activeRole.value?.menuPaths ?? ['/'])



  function canAccess(path: string): boolean {

    return canAccessPath(path, menuPaths.value)

  }



  function canAccessSome(paths: string[]): boolean {

    return paths.some((p) => canAccess(p))

  }



  return { activeRole, menuPaths, canAccess, canAccessSome, roles }

}


