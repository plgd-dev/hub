import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { APPLIED_CONFIGURATIONS_STATUS } from '@/containers/SnippetService/constants'
import { AppliedConfigurationDataType } from '@/containers/SnippetService/ServiceSnippet.types'
import { states } from '@shared-ui/components/Atomic/StatusPill/constants'

export const getConfigurationsPageListI18n = (_: any) => ({
    singleSelected: _(confT.configuration),
    multiSelected: _(confT.configurations),
    tablePlaceholder: _(confT.noConfigurations),
    id: _(g.id),
    name: _(g.name),
    cancel: _(g.cancel),
    action: _(g.action),
    delete: _(g.delete),
    loading: _(g.loading),
    deleteModalSubtitle: _(g.undoneAction),
    view: _(g.view),
    deleteModalTitle: (selected: number) => (selected === 1 ? _(confT.deleteConfigurationMessage) : _(confT.deleteConfigurationsMessage, { count: selected })),
})

export const getAppliedDeviceConfigStatus = (appliedDeviceConfig: AppliedConfigurationDataType) => {
    const statuses = appliedDeviceConfig.resources.map((resource) => {
        if (resource.status === 'PENDING') {
            return 'PENDING'
        }
        return resource.resourceUpdated?.status
    })
    let configStatus = APPLIED_CONFIGURATIONS_STATUS.SUCCESS

    if (statuses.includes('ERROR')) {
        configStatus = APPLIED_CONFIGURATIONS_STATUS.ERROR
    } else if (statuses.includes('PENDING')) {
        configStatus = APPLIED_CONFIGURATIONS_STATUS.PENDING
    }

    return configStatus
}

export const getResourceI18n = (_: any) => ({
    add: _(g.add),
    addContent: _(confT.addContent),
    editContent: _(confT.editContent),
    viewContent: _(confT.viewContent),
    close: _(g.close),
    compactView: _(g.compactView),
    content: _(g.content),
    default: _(g.default),
    duration: _(g.duration),
    edit: _(g.edit),
    fullView: _(g.fullView),
    href: _(g.href),
    name: _(g.name),
    placeholder: _(g.placeholder),
    requiredField: (field: string) => _(g.requiredField, { field }),
    timeToLive: _(g.timeToLive),
    unit: _(g.unit),
    update: _(g.update),
    view: _(g.view),
})

export const getAppliedConfigurationStatusValue = (status: number, _: any) => {
    switch (status) {
        case APPLIED_CONFIGURATIONS_STATUS.ERROR:
            return _(g.error)
        case APPLIED_CONFIGURATIONS_STATUS.PENDING:
            return _(g.pending)
        case APPLIED_CONFIGURATIONS_STATUS.SUCCESS:
        default:
            return _(g.success)
    }
}

export const getAppliedConfigurationStatusStatus = (status: number) => {
    switch (status) {
        case APPLIED_CONFIGURATIONS_STATUS.ERROR:
            return states.OFFLINE
        case APPLIED_CONFIGURATIONS_STATUS.PENDING:
            return states.OCCUPIED
        case APPLIED_CONFIGURATIONS_STATUS.SUCCESS:
        default:
            return states.ONLINE
    }
}
