/**
 * 二次确认删除工具 —— 基于 Element Plus 原生 ElMessageBox.prompt。
 *
 * 弹窗由 element-plus 内部负责渲染：天然在屏幕正中央、z-index / stacking context
 * 与同页 el-drawer / el-dialog 协调，调用方无须关心。
 *
 * 弹窗内提供「📋 快速复制到输入框」按钮，一键把 expected 值填到下方输入框
 * 并触发 element-plus 的 v-model 校验，避免用户手动抄写长串产品名 / ID。
 *
 * 用法：
 *   import { confirmDelete } from '@/utils/confirmDelete'
 *   await confirmDelete({
 *     label: '账号',
 *     expected: row.username,
 *     title: '此操作不可撤销',
 *     description: '将永久删除该员工，并清空其名下关联数据。',
 *   })
 *   await userApi.remove(row.id)
 *
 * 取消 / 校验失败 → reject('canceled')
 * 输入正确 → resolve()
 */
import { h } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 把 text 写到当前打开的 message-box 内的输入框，并触发 element-plus 的 v-model 校验
function fillPromptInput(text) {
  // message-box 栈里取最后一个（同一时刻只有一个 visible）
  const boxes = document.querySelectorAll('.el-message-box')
  const box = boxes[boxes.length - 1]
  if (!box) return false
  const input = box.querySelector('input.el-input__inner')
  if (!input) return false
  // 走原生 setter，绕过 Vue 的 v-model 代理
  const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
  setter.call(input, text)
  input.dispatchEvent(new Event('input', { bubbles: true }))
  return true
}

export function confirmDelete(opts) {
  const {
    label = '确认信息',
    expected,
    title = '此操作不可撤销',
    description = '',
    severity = 'warning',
  } = opts || {}
  const expectedStr = String(expected ?? '').trim()

  // 自定义 message VNode：description + expected 提示 + 快速复制按钮
  const messageVNode = h('div', { class: 'confirm-delete-message' }, [
    description
      ? h('div', { class: 'confirm-delete-desc' }, description)
      : null,
    h(
      'div',
      { class: 'confirm-delete-hint' },
      `为了避免误删，请在下方输入${label}「`
    ),
    h('code', { class: 'confirm-delete-key' }, expectedStr),
    h('span', null, '」以继续：'),
    h(
      'div',
      { class: 'confirm-delete-actions' },
      [
        h(
          'button',
          {
            type: 'button',
            class: 'confirm-delete-copy-btn',
            onClick: () => {
              const ok = fillPromptInput(expectedStr)
              // 双通道：除了直接填进 input，再复制到剪贴板，方便用户 ctrl+v 到别处
              if (navigator.clipboard?.writeText) {
                navigator.clipboard.writeText(expectedStr).catch(() => {})
              }
              ElMessage(ok ? 'success' : 'warning', ok ? '已复制并填入输入框' : '已复制到剪贴板')
            },
          },
          '📋 快速复制到输入框'
        ),
      ]
    ),
  ])

  return ElMessageBox.prompt(messageVNode, title, {
    confirmButtonText: '确认删除',
    cancelButtonText: '取消',
    type: severity,
    inputPlaceholder: `在此输入${label}`,
    // inputValidator 返回 true = 通过；返回 string = 错误提示（按钮 disabled）
    inputValidator: (val) => {
      const v = String(val || '').trim()
      if (!v) return `请输入${label}`
      if (v === expectedStr) return true
      // 数字型 expected 也允许用户只输数字部分（如产品 ID）
      const ev = Number(expectedStr)
      if (expectedStr !== '' && !Number.isNaN(ev) && Number(v) === ev) return true
      return `${label}不正确，请重新输入`
    },
  })
    // 业务侧只关心成功 / 失败；统一转成 reject 形式，便于 `try { await confirmDelete() } catch { return }`
    .then(() => undefined)
    .catch(() => {
      throw new Error('canceled')
    })
}
