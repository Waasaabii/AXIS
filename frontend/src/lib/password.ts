export function validatePasswordPair(password: string, confirmPassword: string) {
  if (password !== confirmPassword) {
    return "两次输入的密码不一致，请重新确认。"
  }
  if (password.length < 6) {
    return "密码至少需要 6 位。"
  }
  return null
}

