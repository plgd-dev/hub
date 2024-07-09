import { AppliedConfigurationDataEnhancedType } from '@/containers/SnippetService/ServiceSnippet.types'
import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

export type Props = {
    data: AppliedConfigurationDataEnhancedType | null
    cancelCommand: (resource: ResourceType) => void
}
