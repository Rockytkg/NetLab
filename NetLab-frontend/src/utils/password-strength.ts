/** 密码规则：8-72 字节，包含大小写字母、数字和特殊字符。 */
export const PASSWORD_MIN_LENGTH = 8
export const PASSWORD_MAX_BYTES = 72

const LOWERCASE_PATTERN = /[a-z]/
const UPPERCASE_PATTERN = /[A-Z]/
const DIGIT_PATTERN = /[0-9]/
const SPECIAL_PATTERN = /[^A-Za-z0-9]/

export interface PasswordStrengthResult {
  length: boolean
  lowercase: boolean
  uppercase: boolean
  digit: boolean
  special: boolean
  acceptable: boolean
}

export function utf8ByteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

export function evaluatePassword(password: string): PasswordStrengthResult {
  const length = Array.from(password).length >= PASSWORD_MIN_LENGTH && utf8ByteLength(password) <= PASSWORD_MAX_BYTES
  const lowercase = LOWERCASE_PATTERN.test(password)
  const uppercase = UPPERCASE_PATTERN.test(password)
  const digit = DIGIT_PATTERN.test(password)
  const special = SPECIAL_PATTERN.test(password)
  return { length, lowercase, uppercase, digit, special, acceptable: length && lowercase && uppercase && digit && special }
}

export function createPasswordStrengthRule(opts: { t: (key: string) => string }) {
  return {
    validator: async (_: unknown, value: string) => {
      if (!value) return Promise.resolve()
      const result = evaluatePassword(value)
      if (!result.length) {
        return Promise.reject(new Error(opts.t('common:passwordStrength.tooShort')))
      }
      if (!result.acceptable) {
        return Promise.reject(new Error(opts.t('common:passwordStrength.requirements')))
      }
      return Promise.resolve()
    },
  }
}
