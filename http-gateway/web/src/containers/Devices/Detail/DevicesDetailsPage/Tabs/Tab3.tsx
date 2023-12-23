import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'

import { Props } from './Tab3.types'
import { messages as t } from '../../../Devices.i18n'
import { messages as g } from '../../../../Global.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'

const Tab3: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { certificates } = props

    if (certificates?.length === 1) {
        const data = certificates[0]

        return (
            <SimpleStripTable
                rows={[
                    {
                        attribute: _(t.certificateName),
                        value: data.commonName,
                    },
                    {
                        attribute: _(g.created),
                        value: data.creationDate ? <DateFormat value={data.creationDate} /> : '-',
                    },
                    {
                        attribute: _(g.expires),
                        value: data.credential.validUntilDate ? <DateFormat value={data.credential.validUntilDate} /> : '-',
                    },
                ]}
            />
        )
    }

    return <div>Certificates table TODO</div>
}

Tab3.displayName = 'Tab3'

export default Tab3
