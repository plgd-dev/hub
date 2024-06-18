import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { APPLIED_DEVICE_CONFIG_STATUS } from '@/containers/SnippetService/constants'

export const getConfigResourcesPageListI18n = (_: any) => ({
    singleSelected: _(confT.resourcesConfiguration),
    multiSelected: _(confT.resourcesConfigurations),
    tablePlaceholder: _(confT.noResourcesConfiguration),
    id: _(g.id),
    name: _(g.name),
    cancel: _(g.cancel),
    action: _(g.action),
    delete: _(g.delete),
    loading: _(g.loading),
    deleteModalSubtitle: _(g.undoneAction),
    view: _(g.view),
    deleteModalTitle: (selected: number) =>
        selected === 1 ? _(confT.deleteResourcesConfigurationMessage) : _(confT.deleteResourcesConfigurationsMessage, { count: selected }),
})

type AppliedDeviceConfigType = {
    resources: {
        correlationId: string
        href: string
        resourceUpdated: {
            auditContext: {
                correlationId: string
                owner: string
            }
            content: string
            status: string
        }
        status: string
    }[]
}

export const getAppliedDeviceConfigStatus = (appliedDeviceConfig: AppliedDeviceConfigType) => {
    const statuses = appliedDeviceConfig.resources.map((resource) => resource.resourceUpdated.status)
    const hasError = statuses.includes('ERROR')
    const hasPending = statuses.includes('PENDING')

    return hasError ? APPLIED_DEVICE_CONFIG_STATUS.ERROR : hasPending ? APPLIED_DEVICE_CONFIG_STATUS.PENDING : APPLIED_DEVICE_CONFIG_STATUS.SUCCESS
}
