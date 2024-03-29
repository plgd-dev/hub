import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import { IconPlus } from '@shared-ui/components/Atomic'
import Button from '@shared-ui/components/Atomic/Button'

import { Props } from './ListHeader.types'
import { messages as t } from '../EnrollmentGroups.i18n'
import { pages } from '@/routes'

const ListHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const navigate = useNavigate()

    return (
        <>
            <Button icon={<IconPlus />} onClick={() => navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.NEW.LINK))} variant='primary'>
                {_(t.enrollmentGroup)}
            </Button>
        </>
    )
}

ListHeader.displayName = 'ListHeader'

export default ListHeader
