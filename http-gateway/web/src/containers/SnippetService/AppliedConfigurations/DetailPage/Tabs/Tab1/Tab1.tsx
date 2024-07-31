import React, { FC } from 'react'
import { generatePath, useNavigate } from 'react-router-dom'
import { useIntl } from 'react-intl'

import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Tag from '@shared-ui/components/Atomic/Tag'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'

import { Props } from './Tab1.types'
import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import testId from '@/testId'

const Tab1: FC<Props> = (props) => {
    const { data } = props

    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()

    return (
        <SimpleStripTable
            leftColSize={6}
            rightColSize={6}
            rows={[
                {
                    attribute: _(g.deviceName),
                    value: (
                        <FormGroup id='name' marginBottom={false} style={{ width: '100%' }}>
                            <FormInput disabled align='right' size='small' value={data?.name} />
                        </FormGroup>
                    ),
                },
                {
                    attribute: _(confT.configuration),
                    value: (
                        <Tag
                            dataTestId={testId.snippetService.appliedConfigurations.detail.tab1.configurationLink}
                            onClick={() =>
                                navigate(
                                    generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, {
                                        configurationId: data?.configurationId?.id,
                                        tab: '',
                                    })
                                )
                            }
                            variant={tagVariants.BLUE}
                        >
                            <IconLink />
                            <Spacer type='ml-2'>{data?.configurationName}</Spacer>
                            <Spacer type='ml-2'>(v.{data?.configurationId.version})</Spacer>
                        </Tag>
                    ),
                },
                {
                    attribute: _(confT.condition),
                    value: data?.conditionId ? (
                        <Tag
                            dataTestId={testId.snippetService.appliedConfigurations.detail.tab1.conditionLink}
                            onClick={() =>
                                navigate(
                                    generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, {
                                        conditionId: data?.conditionId?.id,
                                        tab: '',
                                    })
                                )
                            }
                            variant={tagVariants.BLUE}
                        >
                            <IconLink />
                            <Spacer type='ml-2'>{data?.conditionName}</Spacer>
                            <Spacer type='ml-2'>(v.{data?.conditionId.version})</Spacer>
                        </Tag>
                    ) : (
                        <StatusTag variant={statusTagVariants.NORMAL}>{_(confT.onDemand)}</StatusTag>
                    ),
                },
            ]}
        />
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
