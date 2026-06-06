import type { AuthUser } from '@/entities/user/model/user'

import { getAxiosInstance } from './request'

export async function getProfile(): Promise<AuthUser> {
  const response = await getAxiosInstance().get<AuthUser>('/auth/profile')
  return response.data
}

export async function login(input: { username: string; password: string }): Promise<AuthUser> {
  const response = await getAxiosInstance().post<AuthUser>('/auth/login', input)
  return response.data
}

export async function logout(): Promise<void> {
  await getAxiosInstance().post('/auth/logout')
}
