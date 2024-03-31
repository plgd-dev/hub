import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import Column from '@shared-ui/components/Atomic/Grid/Column'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TagGroup, { justifyContent } from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import TileExpand from '@shared-ui/components/Atomic/TileExpand/TileExpand'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import { Information } from '@shared-ui/components/Atomic/TileExpand/TileExpand.types'
import Row from '@shared-ui/components/Atomic/Grid/Row'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../../ProvisioningRecords.i18n'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import { Props } from './Tab1.types'
import TileExpandEnhanced from '../../TileExpandEnhanced/TileExpandEnhanced'
import { useMediaQuery } from 'react-responsive'
import Spacer from '@shared-ui/components/Atomic/Spacer'

const Tab1: FC<Props> = (props) => {
    const { data, isDeviceMode } = props

    const { formatMessage: _ } = useIntl()
    const useSpace = useMediaQuery({
        query: '(max-width: 1399px)',
    })

    return (
        <div style={{ width: '100%', overflow: 'hidden' }}>
            <Row>
                <Column xxl={isDeviceMode ? 12 : 6}>
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
                                            {data.localEndpoints?.map?.((endpoint: string) => <Tag key={endpoint}>{endpoint}</Tag>)}
                                        </TagGroup>
                                    ) : (
                                        '-'
                                    ),
                                },
                            ]}
                        />
                    )}
                </Column>
                <Column xxl={1} />
                <Column xxl={isDeviceMode ? 12 : 5}>
                    <Spacer type={useSpace || isDeviceMode ? 'mt-4' : 'pb-2'}>
                        <Row>
                            <Column xl={isDeviceMode ? 4 : 12}>
                                {data?.cloud && (
                                    <TileExpandEnhanced
                                        data={data.cloud}
                                        information={{
                                            groupTitle: _(g.information),
                                            rows: [
                                                {
                                                    attribute: _(t.deviceGateways),
                                                    value: data.cloud.gateways ? (
                                                        <TagGroup
                                                            i18n={{
                                                                more: _(g.more),
                                                                modalHeadline: _(t.deviceGateways),
                                                            }}
                                                            justifyContent={justifyContent.END}
                                                        >
                                                            {data.cloud.gateways?.map?.((gateway: { uri: string; id: string }, key: number) => (
                                                                <Tag
                                                                    key={gateway.id}
                                                                    variant={key === data.cloud.selectedGateway ? tagVariants.BLUE : tagVariants.WHITE}
                                                                >
                                                                    {gateway.uri}
                                                                </Tag>
                                                            ))}
                                                        </TagGroup>
                                                    ) : (
                                                        '-'
                                                    ),
                                                    copyValue: data.cloud.gateways?.map?.((gateway: { uri: string; id: string }) => gateway.uri).join(' '),
                                                },
                                                {
                                                    attribute: _(t.provider),
                                                    value: data.cloud.providerName,
                                                },
                                                data.cloud.id
                                                    ? {
                                                          attribute: _(g.id),
                                                          value: data.cloud.id || '-',
                                                      }
                                                    : undefined,
                                            ].filter((i) => !!i) as Information[],
                                        }}
                                        title={_(t.cloud)}
                                    />
                                )}
                            </Column>
                            <Column xl={isDeviceMode ? 4 : 12}>
                                {data?.ownership && (
                                    <TileExpandEnhanced
                                        data={data.ownership}
                                        information={{
                                            groupTitle: _(g.information),
                                            rows: [
                                                {
                                                    attribute: _(t.ownerId),
                                                    value: data.ownership.owner,
                                                },
                                            ],
                                        }}
                                        title={_(t.ownership)}
                                    />
                                )}
                            </Column>
                            <Column xl={isDeviceMode ? 4 : 12}>
                                {data?.plgdTime && (
                                    <TileExpand
                                        hasExpand={false}
                                        i18n={{
                                            copy: _(g.copy),
                                        }}
                                        time={data.plgdTime.date ? <DateFormat value={data.plgdTime.date} /> : '-'}
                                        title={_(t.timeSynchronisation)}
                                    />
                                )}
                            </Column>
                        </Row>
                    </Spacer>
                </Column>
            </Row>
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
