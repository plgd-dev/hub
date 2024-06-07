import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'

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
