import React, { FC, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'

import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TileExpand from '@shared-ui/components/Atomic/TileExpand/TileExpand'

import { messages as g } from '@/containers/Global.i18n'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../ProvisioningRecords.i18n'
import { useProvisioningRecordsDetail } from '../../hooks'
import DateFormat from '@/containers/PendingCommands/DateFormat'
import * as styles from './ProvisioningRecordsDetailPage.styles'
import { getStatusFromCode } from '../../utils'
import { TileExpandEnhancedType } from '../ListPage/ProvisioningRecordsListPage.types'
import DetailHeader from '../DetailHeader/DetailHeader'
import PageLayout from '@/containers/Common/PageLayout'

const TileExpandEnhanced: FC<TileExpandEnhancedType> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { data, information, title } = props
    return (
        <TileExpand
            css={styles.listItem}
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
    )
}

const ProvisioningRecordsListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const { recordId } = useParams()

    const { data, loading, error, refresh } = useProvisioningRecordsDetail(recordId)

    const isOnline = true

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(t.provisioningRecords), link: '/device-provisioning/provisioning-records' },
            { label: data?.enrollmentGroupData?.name! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data?.enrollmentGroupData]
    )

    if (error) {
        return <div>{error}</div>
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <DetailHeader enrollmentGroupData={data?.enrollmentGroupData} enrollmentGroupId={data?.enrollmentGroupId} id={recordId} refresh={refresh} />
            }
            headlineStatusTag={<StatusTag variant={isOnline ? 'success' : 'error'}>{isOnline ? _(g.online) : _(g.offline)}</StatusTag>}
            loading={loading}
            title={data?.enrollmentGroupData?.name || '-'}
        >
            {!!data && (
                <Row>
                    <Column xl={6}>
                        {data && (
                            <SimpleStripTable
                                leftColSize={4}
                                rightColSize={8}
                                rows={[
                                    { attribute: _(g.id), value: data.id },
                                    { attribute: _(t.deviceID), value: data.deviceId },
                                    { attribute: _(t.enrollmentGroupId), value: data.enrollmentGroupId },
                                    { attribute: _(t.firstAttestation), value: data.creationDate ? <DateFormat value={data.creationDate} /> : '-' },
                                    { attribute: _(t.latestAttestation), value: data.attestation?.date ? <DateFormat value={data.attestation.date} /> : '-' },
                                ]}
                            />
                        )}
                    </Column>
                    <Column xl={1} />
                    <Column xl={5}>
                        {data?.credential && <TileExpandEnhanced data={data.credential} title={_(t.credentials)} />}
                        {data?.acl && (
                            <TileExpand
                                css={styles.listItem}
                                i18n={{
                                    copy: _(g.copy),
                                }}
                                statusTag={
                                    <StatusTag variant={getStatusFromCode(data.acl.status.coapCode)}>{getStatusFromCode(data.acl.status.coapCode)}</StatusTag>
                                }
                                time={data.acl.status.date ? <DateFormat value={data.acl.status.date} /> : '-'}
                                title={_(t.acls)}
                            />
                        )}
                        {data?.cloud && (
                            <TileExpandEnhanced
                                data={data.cloud}
                                information={{
                                    groupTitle: _(g.information),
                                    rows: [
                                        {
                                            attribute: _(t.coapGateway),
                                            value: data.cloud.coapGateway,
                                        },
                                        {
                                            attribute: _(t.provider),
                                            value: data.cloud.providerName,
                                        },
                                        {
                                            attribute: _(g.id),
                                            value: data.cloud.id,
                                        },
                                    ],
                                }}
                                title={_(t.cloud)}
                            />
                        )}
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
                        {data?.plgdTime && (
                            <TileExpand
                                css={styles.listItem}
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
            )}
        </PageLayout>
    )
}

ProvisioningRecordsListPage.displayName = 'ProvisioningRecordsListPage'

export default ProvisioningRecordsListPage
