import { defineStore } from 'pinia'

export const useWorkflowStore = defineStore('workflow', {
  state: () => ({
    recognized: [] as string[],
  }),
  actions: {
    setRecognized(values: string[]) {
      this.recognized = values
    },
  },
})
