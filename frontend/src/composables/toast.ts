import { reactive } from 'vue'

export interface Toast {
  id: number
  text: string
  kind: 'success' | 'error' | 'info'
}

let seq = 0
export const toasts = reactive<Toast[]>([])

function push(text: string, kind: Toast['kind'] = 'info') {
  const id = ++seq
  toasts.push({ id, text, kind })
  window.setTimeout(() => {
    const i = toasts.findIndex((t) => t.id === id)
    if (i >= 0) toasts.splice(i, 1)
  }, 4000)
}

export function useToast() {
  return { push }
}
