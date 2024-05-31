import React, { forwardRef } from 'react'
import { useIntl } from 'react-intl'

import TileExpand from '@shared-ui/components/Atomic/TileExpand/TileExpand'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { TileExpandEnhancedType } from './TileExpandEnhanced.types'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecords.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import ConditionalWrapper from '@shared-ui/components/Atomic/ConditionalWrapper'

const TileExpandEnhanced = forwardRef<HTMLDivElement, TileExpandEnhancedType>((props, ref) => {
    const { formatMessage: _ } = useIntl()
    const { data, divWrapper, information, noSpace, title, ...rest } = props

    return (
        <ConditionalWrapper
            condition={!!ref || !noSpace || divWrapper}
            wrapper={(children) => {
                if (divWrapper) {
                    return <div>{children}</div>
                } else {
                    return (
                        <Spacer ref={ref} type='mb-2'>
                            {children}
                        </Spacer>
                    )
                }
            }}
        >
            <TileExpand
                {...rest}
                error={
                    data.status.coapCode === 0 && data.status.errorMessage
                        ? {
                              groupTitle: _(g.information),
                              message: data.status.errorMessage,
                              title: _(t.cannotProvision, { variant: title }),
                          }
                        : undefined
                }
                hasExpand={!!information}
                i18n={{
                    copy: _(g.copy),
                }}
                information={information}
                statusTag={<StatusTag variant={getStatusFromCode(data.status.coapCode)}>{getStatusFromCode(data.status.coapCode)}</StatusTag>}
                time={data.status.date ? <DateFormat value={data.status.date} /> : '-'}
                title={title}
            />
        </ConditionalWrapper>
    )
})

TileExpandEnhanced.displayName = 'TileExpandEnhanced'

export default TileExpandEnhanced
