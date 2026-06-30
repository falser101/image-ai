/**
 * 实时查重 composable：复用 caller 的 ref 自动 watch，输入变化时 debounce 调后端。
 * 返回状态：
 *   null       = 还没开始校验（输入空时）
 *   'checking' = 校验中
 *   'ok'       = 名称可用
 *   'duplicate'= 名称被占用
 *   'invalid'  = 长度等不符合规则
 *   'error'    = 网络/服务端错误
 *
 * 用法：
 *   const name = ref('')
 *   const checker = useNameChecker(name, (s) => productApi.checkName(s).then(r => r.data))
 *   <el-input v-model="name" :status="checker.inputStatus()" />
 *   <span v-if="checker.message.value">{{ checker.message.value }}</span>
 *   <el-button :disabled="!checker.canSubmit()">提交</el-button>
 */
import { ref, watch } from 'vue'

export function useNameChecker(nameRef, checkFn, opts = {}) {
  const debounceMs = opts.debounceMs ?? 300
  const state = ref(null)
  const message = ref('')
  let timer = null
  let abortCtrl = null
  let lastReqId = 0

  const recheck = (val) => {
    const trimmed = String(val || '').trim()
    if (timer) clearTimeout(timer)
    if (!trimmed) {
      state.value = null
      message.value = ''
      return
    }
    state.value = 'checking'
    message.value = '校验中…'
    timer = setTimeout(async () => {
      const reqId = ++lastReqId
      try {
        if (abortCtrl) abortCtrl.abort()
        abortCtrl = new AbortController()
        const r = await checkFn(trimmed, { signal: abortCtrl.signal })
        if (reqId !== lastReqId) return
        if (r.valid === false) {
          state.value = 'invalid'
          message.value = r.reason || '名称不符合规则'
        } else if (r.exists) {
          state.value = 'duplicate'
          message.value = `已有同名产品：「${trimmed}」`
        } else {
          state.value = 'ok'
          message.value = '名称可用'
        }
      } catch (e) {
        if (e?.name === 'AbortError' || e?.name === 'CanceledError') return
        if (reqId !== lastReqId) return
        state.value = 'error'
        message.value = e?.message || '校验失败，请稍后重试'
      }
    }, debounceMs)
  }

  watch(nameRef, recheck)

  const inputStatus = () => {
    switch (state.value) {
      case 'ok': return 'success'
      case 'duplicate':
      case 'invalid': return 'error'
      case 'checking': return 'warning'
      default: return ''
    }
  }
  const canSubmit = () => state.value === 'ok'

  return { state, message, inputStatus, canSubmit }
}

