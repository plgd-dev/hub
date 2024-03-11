import React, { FC, useCallback, useMemo } from 'react'
import { useIntl } from 'react-intl'

import { isOwnerId, isYourId } from '@shared-ui/common/services/api-utils'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../ProvisioningRecords.i18n'
import { Props } from './SubjectColumn.types'

const SubjectColumn: FC<Props> = (props) => {
    const { value, owner, hubsData, hubId } = props

    const { formatMessage: _ } = useIntl()

    const isHub = useMemo(() => hubId && value === hubId, [value, hubId])
    const isYour = useMemo(() => hubsData && isYourId(value, hubsData), [value, hubsData])
    const isOwner = useMemo(() => owner && isOwnerId(value, owner), [value, owner])

    const baseTag = useCallback(
        (text: string) => (
            <Spacer style={{ display: 'inline-flex' }} type='ml-2'>
                <StatusTag lowercase={false} variant='info'>
                    {text}
                </StatusTag>
            </Spacer>
        ),
        []
    )

    return (
        <span className='no-wrap-text'>
            {value}
            {isHub && baseTag(_(g.hubId))}
            {isYour && baseTag(_(t.yourId))}
            {isOwner && baseTag(_(t.ownerId))}
        </span>
    )
}

SubjectColumn.displayName = 'SubjectColumn'

export default SubjectColumn
