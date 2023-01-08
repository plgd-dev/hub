import { fetchApi } from '@shared-ui/common/services'

export const getAppWellKnownConfiguration = (wellKnowConfigUrl: string) =>
  fetchApi(`${wellKnowConfigUrl}/.well-known/configuration`, {
    useToken: false,
  })
