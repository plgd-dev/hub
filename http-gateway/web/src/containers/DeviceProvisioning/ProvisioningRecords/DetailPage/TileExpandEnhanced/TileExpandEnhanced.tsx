import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import TileExpand from '@shared-ui/components/Atomic/TileExpand/TileExpand'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { TileExpandEnhancedType } from './TileExpandEnhanced.types'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecords.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import DateFormat from '@/containers/PendingCommands/DateFormat'

const TileExpandEnhanced: FC<TileExpandEnhancedType> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { data, information, title } = props
    return (
        <Spacer type='mb-2'>
            <TileExpand
                error={
                    data.status.coapCode === 0 && data.status.errorMessage
                        ? {
                              groupTitle: _(g.information),
                              message: data.status.errorMessage,
                              title: _(t.cannotProvision, { variant: title }),
                          }
                        : undefined
                }
                i18n={{
                    copy: _(g.copy),
                }}
                information={information}
                statusTag={<StatusTag variant={getStatusFromCode(data.status.coapCode)}>{getStatusFromCode(data.status.coapCode)}</StatusTag>}
                time={data.status.date ? <DateFormat value={data.status.date} /> : '-'}
                title={title}
            />
        </Spacer>
    )
}

TileExpandEnhanced.displayName = 'TileExpandEnhanced'

export default TileExpandEnhanced
