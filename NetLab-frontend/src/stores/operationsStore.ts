import { create } from 'zustand'
import type { DeviceGroup, ManagedDevice } from '@/types/operations'

/**
 * 运维数据 store（占位）：Phase 2/3 接入真实设备清单与监控数据之前，
 * 仪表盘统计从这里读取（当前恒为空数组）。
 */
interface OperationsState {
  deviceGroups: DeviceGroup[]
  devices: ManagedDevice[]
}

export const useOperationsStore = create<OperationsState>()(() => ({
  deviceGroups: [],
  devices: [],
}))
