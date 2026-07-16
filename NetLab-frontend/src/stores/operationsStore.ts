import { create } from 'zustand'
import type { DeviceGroup, ManagedDevice, OperationsFilter } from '@/types/operations'

interface OperationsState {
  deviceGroups: DeviceGroup[]
  devices: ManagedDevice[]
  filter: OperationsFilter
  selectedGroupIds: string[]
  activeGroupId: string | null
  activeDeviceId: string | null

  setDeviceGroups: (deviceGroups: DeviceGroup[]) => void
  setDevices: (devices: ManagedDevice[]) => void
  setFilter: (filter: Partial<OperationsFilter>) => void
  setSelectedGroupIds: (ids: string[]) => void
  setActiveGroupId: (id: string | null) => void
  setActiveDeviceId: (id: string | null) => void
  updateDeviceGroup: (id: string, updates: Partial<DeviceGroup>) => void
  removeDeviceGroup: (id: string) => void
}

export const useOperationsStore = create<OperationsState>()((set) => ({
  deviceGroups: [],
  devices: [],
  filter: { status: null, search: '' },
  selectedGroupIds: [],
  activeGroupId: null,
  activeDeviceId: null,

  setDeviceGroups: (deviceGroups) => set({ deviceGroups }),
  setDevices: (devices) => set({ devices }),

  setFilter: (filter) =>
    set((state) => ({
      filter: { ...state.filter, ...filter },
    })),

  setSelectedGroupIds: (selectedGroupIds) => set({ selectedGroupIds }),
  setActiveGroupId: (activeGroupId) => set({ activeGroupId }),
  setActiveDeviceId: (activeDeviceId) => set({ activeDeviceId }),

  updateDeviceGroup: (id, updates) =>
    set((state) => ({
      deviceGroups: state.deviceGroups.map((group) =>
        group.id === id ? { ...group, ...updates } : group
      ),
    })),

  removeDeviceGroup: (id) =>
    set((state) => ({
      deviceGroups: state.deviceGroups.filter((group) => group.id !== id),
      selectedGroupIds: state.selectedGroupIds.filter((selectedId) => selectedId !== id),
    })),
}))
