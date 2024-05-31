import { fetchApi, security } from '@shared-ui/common/services'
import { SecurityConfig } from '@/containers/App/App.types'
import { GITHUB_VERSION_URL } from '@/constants'

export const getAppWellKnownConfiguration = (wellKnowConfigUrl: string) => {
    const { cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig

    return fetchApi(`${wellKnowConfigUrl}/.well-known/configuration`, {
        useToken: false,
        cancelRequestDeadlineTimeout,
    })
}

export const getVersionNumberFromGithub = () => fetchApi(GITHUB_VERSION_URL, { useToken: false })
