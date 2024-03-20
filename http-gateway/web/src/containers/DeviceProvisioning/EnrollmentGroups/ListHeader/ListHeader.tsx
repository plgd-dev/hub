import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate } from 'react-router-dom'

import { IconPlus } from '@shared-ui/components/Atomic'
import Button from '@shared-ui/components/Atomic/Button'

import { Props } from './ListHeader.types'
import { messages as t } from '../EnrollmentGroups.i18n'

const ListHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const navigate = useNavigate()

    return (
        <>
            <Button icon={<IconPlus />} onClick={() => navigate('/device-provisioning/enrollment-groups/new-enrollment-group')} variant='primary'>
                {_(t.enrollmentGroup)}
            </Button>
        </>
    )
}

ListHeader.displayName = 'ListHeader'

export default ListHeader
