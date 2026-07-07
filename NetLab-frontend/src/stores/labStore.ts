import { create } from 'zustand'
import type { LabItem, LabFilter } from '@/types/lab'

interface LabState {
  /** 实验室列表 */
  labs: LabItem[]
  /** 筛选条件 */
  filter: LabFilter
  /** 当前选中的实验室 ID 列表 */
  selectedLabIds: string[]
  /** 当前激活的实验室 ID（正在编辑/监控的） */
  activeLabId: string | null

  /** 设置实验室列表 */
  setLabs: (labs: LabItem[]) => void
  /** 更新筛选条件 */
  setFilter: (filter: Partial<LabFilter>) => void
  /** 设置选中实验室 */
  setSelectedLabIds: (ids: string[]) => void
  /** 设置当前激活实验室 */
  setActiveLabId: (id: string | null) => void
  /** 更新单个实验室 */
  updateLab: (id: string, updates: Partial<LabItem>) => void
  /** 删除实验室 */
  removeLab: (id: string) => void
}

export const useLabStore = create<LabState>()((set) => ({
  labs: [],
  filter: { status: null, search: '' },
  selectedLabIds: [],
  activeLabId: null,

  setLabs: (labs) => set({ labs }),

  setFilter: (filter) =>
    set((state) => ({
      filter: { ...state.filter, ...filter },
    })),

  setSelectedLabIds: (selectedLabIds) => set({ selectedLabIds }),

  setActiveLabId: (activeLabId) => set({ activeLabId }),

  updateLab: (id, updates) =>
    set((state) => ({
      labs: state.labs.map((lab) =>
        lab.id === id ? { ...lab, ...updates } : lab
      ),
    })),

  removeLab: (id) =>
    set((state) => ({
      labs: state.labs.filter((lab) => lab.id !== id),
      selectedLabIds: state.selectedLabIds.filter((sid) => sid !== id),
    })),
}))
