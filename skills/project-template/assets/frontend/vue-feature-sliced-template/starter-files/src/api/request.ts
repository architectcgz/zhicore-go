import axios, { AxiosError } from 'axios'

export class ApiError extends Error {
  status?: number
  requestUrl?: string

  constructor(message: string, options: { status?: number; requestUrl?: string } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = options.status
    this.requestUrl = options.requestUrl
  }
}

const axiosInstance = axios.create({
  baseURL: '/api',
  withCredentials: true,
})

axiosInstance.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ message?: string }>) => {
    const status = error.response?.status
    const requestUrl = error.config?.url
    const message = error.response?.data?.message || error.message || '请求失败'
    return Promise.reject(new ApiError(message, { status, requestUrl }))
  }
)

export function getAxiosInstance() {
  return axiosInstance
}
