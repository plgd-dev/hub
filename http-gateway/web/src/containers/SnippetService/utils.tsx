import cloneDeep from 'lodash/cloneDeep'
import isEmpty from 'lodash/isEmpty'

import { states } from '@shared-ui/components/Atomic/StatusPill/constants'
import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'
import { getResourceStatus } from '@shared-ui/components/Organisms/ResourceToggleCreator'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { APPLIED_CONFIGURATIONS_STATUS } from '@/containers/SnippetService/constants'
import { AppliedConfigurationDataType, AppliedConfigurationStatusType } from '@/containers/SnippetService/ServiceSnippet.types'

export const getConfigurationsPageListI18n = (_: any) => ({
    singleSelected: _(confT.configuration),
    multiSelected: _(confT.configurations),
    tablePlaceholder: _(confT.noConfigurations),
    id: _(g.id),
    name: _(g.name),
    cancel: _(g.cancel),
    action: _(g.action),
    invoke: _(g.invoke),
    delete: _(g.delete),
    loading: _(g.loading),
    deleteModalSubtitle: _(g.undoneAction),
    view: _(g.view),
    deleteModalTitle: (selected: number) => (selected === 1 ? _(confT.deleteConfigurationMessage) : _(confT.deleteConfigurationsMessage, { count: selected })),
})

export const getAppliedDeviceConfigStatus = (appliedDeviceConfig: AppliedConfigurationDataType) => {
    const statuses = appliedDeviceConfig.resources.map((resource) => getResourceStatus(resource))

    if (statuses.includes('ERROR')) {
        return APPLIED_CONFIGURATIONS_STATUS.ERROR
    } else if (statuses.includes('TIMEOUT')) {
        return APPLIED_CONFIGURATIONS_STATUS.TIMEOUT
    } else if (statuses.includes('PENDING')) {
        return APPLIED_CONFIGURATIONS_STATUS.PENDING
    } else if (statuses.includes('CANCELED')) {
        return APPLIED_CONFIGURATIONS_STATUS.CANCELED
    }

    return APPLIED_CONFIGURATIONS_STATUS.OK
}

export const getResourceI18n = (_: any) => ({
    add: _(g.add),
    addContent: _(confT.addContent),
    editContent: _(confT.editContent),
    viewContent: _(confT.viewContent),
    cancel: _(g.cancel),
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

export const getAppliedConfigurationStatusValue = (status: AppliedConfigurationStatusType, _: any) => {
    switch (status) {
        case APPLIED_CONFIGURATIONS_STATUS.ERROR:
            return _(g.error)
        case APPLIED_CONFIGURATIONS_STATUS.CANCELED:
            return _(g.canceled)
        case APPLIED_CONFIGURATIONS_STATUS.TIMEOUT:
            return _(g.timeout)
        case APPLIED_CONFIGURATIONS_STATUS.PENDING:
            return _(g.pending)
        case APPLIED_CONFIGURATIONS_STATUS.OK:
        default:
            return _(g.success)
    }
}

export const getAppliedConfigurationStatusStatus = (status: AppliedConfigurationStatusType) => {
    switch (status) {
        case APPLIED_CONFIGURATIONS_STATUS.ERROR:
        case APPLIED_CONFIGURATIONS_STATUS.TIMEOUT:
            return states.OFFLINE
        case APPLIED_CONFIGURATIONS_STATUS.PENDING:
        case APPLIED_CONFIGURATIONS_STATUS.CANCELED:
            return states.OCCUPIED
        case APPLIED_CONFIGURATIONS_STATUS.OK:
        default:
            return states.ONLINE
    }
}

export const formatConfigurationResources = (data: any) => {
    const dataForSave = cloneDeep(data)
    delete dataForSave?.id

    if (dataForSave.resources) {
        dataForSave.resources = dataForSave.resources.map((resource: ResourceType) => ({
            ...resource,
            content: {
                data: btoa(resource.content.toString()),
                contentType: 'application/json',
                coapContentFormat: -1,
            },
        }))
    } else {
        dataForSave.resources = []
    }

    return dataForSave
}

export const hasInvalidConfigurationResource = (resources: ResourceType[]) =>
    resources?.some(
        (resource) =>
            resource.href === '' ||
            resource.timeToLive === '' ||
            resource.content === '{}' ||
            (typeof resource.content === 'object' ? isEmpty(resource.content) : !resource.content)
    )

export const hasConfigurationResourceError = (resources: ResourceType[]) => {
    const hrefs: string[] = []
    let er = false

    if (resources.length === 0) {
        return true
    }

    resources.forEach((resource, index) => {
        if (hrefs.includes(resource.href)) {
            er = true
        }
        hrefs.push(resource.href)
    })

    return er
}
