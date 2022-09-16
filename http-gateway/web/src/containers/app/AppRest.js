import { fetchApi } from '@/common/services'

export const getAppWellKnownConfiguration = wellKnowConfigUrl =>
  fetchApi(`${wellKnowConfigUrl}/.well-known/configuration`, {
    useToken: false,
  })
