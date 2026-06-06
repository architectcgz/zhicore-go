export type UserRole = 'admin' | 'teacher' | 'student'

export interface AuthUser {
  id: string
  username: string
  role: UserRole
  displayName?: string
}

export function getUserDisplayName(user: AuthUser | null): string {
  if (!user) {
    return '未登录'
  }
  return user.displayName || user.username
}
