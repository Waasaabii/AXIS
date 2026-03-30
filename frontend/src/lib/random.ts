export function randomString(length: number): string {
  const chars = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789'
  let result = ''
  const array = new Uint32Array(length)
  crypto.getRandomValues(array)
  for (let i = 0; i < length; i++) result += chars[array[i] % chars.length]
  return result
}

export function generateUsername(): string {
  return 'user' + randomString(6)
}

export function generatePassword(): string {
  return randomString(16)
}
