export * from "./provider-types"

import { httpAPI } from "./http-provider"
import type { AxisAPI } from "./provider-types"
import { hasWailsRuntime, wailsAPI } from "./wails-provider"

function currentAPI(): AxisAPI {
  return hasWailsRuntime() ? wailsAPI : httpAPI
}

export const api: AxisAPI = new Proxy({} as AxisAPI, {
  get(_target, property) {
    return (...args: unknown[]) => {
      const method = currentAPI()[property as keyof AxisAPI]
      return (method as (...methodArgs: unknown[]) => unknown)(...args)
    }
  },
})
