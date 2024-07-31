import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'

import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import ResourceToggleCreator, { getResourceStatus, getResourceStatusTag } from '@shared-ui/components/Organisms/ResourceToggleCreator'

import { Props } from './Tab2.types'
import { getResourceI18n } from '@/containers/SnippetService/utils'
import testId from '@/testId'

const Tab2: FC<Props> = (props) => {
    const { data, cancelCommand } = props
    const { formatMessage: _ } = useIntl()

    const resourceI18n = useMemo(() => getResourceI18n(_), [_])

    return (
        <div>
            {data?.resources &&
                data?.resources?.map((resource: ResourceType, key: number) => (
                    <Spacer key={resource.href} type='mb-2'>
                        <ResourceToggleCreator
                            defaultOpen
                            readOnly
                            dataTestId={`${testId.snippetService.appliedConfigurations.detail.tab2.resourceToggleCreator}-${key}`}
                            i18n={resourceI18n}
                            onCancelPending={getResourceStatus(resource) === 'PENDING' ? (resource) => cancelCommand(resource) : undefined}
                            resourceData={resource}
                            statusTag={getResourceStatusTag(resource)}
                        />
                    </Spacer>
                ))}
        </div>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
