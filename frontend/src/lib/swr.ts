export type MutatorFn = () => Promise<unknown> | unknown

export function mutateMany(...mutators: MutatorFn[]) {
  return Promise.all(mutators.map((mutate) => mutate())).then(() => undefined)
}

export function mutateKeys(mutate: (key: string) => Promise<unknown>, keys: readonly string[]) {
  return Promise.all(keys.map((key) => mutate(key))).then(() => undefined)
}

