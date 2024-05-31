import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import TagGroup, { justifyContent } from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import { Information } from '@shared-ui/components/Atomic/TileExpand/TileExpand.types'
import TileExpand from '@shared-ui/components/Atomic/TileExpand/TileExpand'

import TileExpandEnhanced from '@/containers/DeviceProvisioning/ProvisioningRecords/DetailPage/TileExpandEnhanced'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecords.i18n'
import * as styles from './OverviewRow.styles'
import { Props } from './OverviewRow.types'
import DateFormat from '@/containers/PendingCommands/DateFormat'

const OverviewRow: FC<Props> = (props) => {
    const { data } = props

    const { formatMessage: _ } = useIntl()

    return (
        <div css={styles.row}>
            {data?.cloud && (
                <TileExpandEnhanced
                    divWrapper
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
                                                key={`${gateway.uri}${gateway.id}}`}
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
            {data?.ownership && (
                <TileExpandEnhanced
                    divWrapper
                    data={data.ownership}
                    information={{
                        groupTitle: _(g.information),
                        rows: [
                            {
                                attribute: _(g.owner),
                                value: data.ownership.owner,
                            },
                        ],
                    }}
                    title={_(t.ownership)}
                />
            )}
            {data?.plgdTime && (
                <div>
                    <TileExpand
                        hasExpand={false}
                        i18n={{
                            copy: _(g.copy),
                        }}
                        time={data.plgdTime.date ? <DateFormat value={data.plgdTime.date} /> : '-'}
                        title={_(t.timeSynchronisation)}
                    />
                </div>
            )}
            {data?.credential && <TileExpandEnhanced divWrapper data={data.credential} title={_(t.credentials)} />}
            {data?.acl && <TileExpandEnhanced divWrapper data={data.acl} title={_(t.acls)} />}
        </div>
    )
}

OverviewRow.displayName = 'OverviewRow'

export default OverviewRow
