import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { useMediaQuery } from 'react-responsive'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TagGroup, { justifyContent } from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../../ProvisioningRecords.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { Props } from './Tab1.types'

const Tab1: FC<Props> = (props) => {
    const { data, isDeviceMode, refs } = props

    const { formatMessage: _ } = useIntl()
    const useSpace = useMediaQuery({
        query: '(max-width: 1399px)',
    })

    return (
        <div style={{ width: '100%', overflow: 'hidden' }}>
            {data && (
                <SimpleStripTable
                    leftColSize={4}
                    rightColSize={8}
                    rows={[
                        { attribute: _(t.provisioningRecordId), value: data.id },
                        { attribute: _(t.deviceID), value: data.deviceId },
                        { attribute: _(t.enrollmentGroupId), value: data.enrollmentGroupId },
                        { attribute: _(t.firstAttestation), value: data.creationDate ? <DateFormat value={data.creationDate} /> : '-' },
                        { attribute: _(t.latestAttestation), value: data.attestation?.date ? <DateFormat value={data.attestation.date} /> : '-' },
                        {
                            attribute: _(t.endpoints),
                            value: data.localEndpoints ? (
                                <TagGroup
                                    i18n={{
                                        more: _(g.more),
                                        modalHeadline: _(t.endpoints),
                                    }}
                                    justifyContent={justifyContent.END}
                                >
                                    {data.localEndpoints?.map?.((endpoint: string, key) => <Tag key={`${endpoint}${key}`}>{endpoint}</Tag>)}
                                </TagGroup>
                            ) : (
                                '-'
                            ),
                        },
                    ]}
                />
            )}
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
