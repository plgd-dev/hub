import React, { FC, useState } from 'react'
import TileToggleRow from '@shared-ui/components/new/TileToggle/TileToggleRow'
import TileToggle from '@shared-ui/components/new/TileToggle'
import { Props } from './Tab1.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { useIntl } from 'react-intl'
import SimpleStripTable from '@shared-ui/components/new/SimpleStripTable'
import TagGroup from '@shared-ui/components/new/TagGroup'
import Tag from '@shared-ui/components/new/Tag'
import { Icon } from '@shared-ui/components/new/Icon'

const Tab1: FC<Props> = (props) => {
    const { isTwinEnabled, setTwinSynchronization, twinSyncLoading, deviceId, types } = props
    const { formatMessage: _ } = useIntl()
    const [state, setState] = useState({
        tile2: false,
        tile3: true,
    })
    return (
        <div
            style={{
                paddingTop: 8,
            }}
        >
            <TileToggleRow>
                <TileToggle checked={isTwinEnabled} loading={twinSyncLoading} name={_(t.twinState)} onChange={() => setTwinSynchronization(!isTwinEnabled)} />
                <TileToggle checked={state.tile2} name={_(t.subscribeNotify)} onChange={() => setState({ ...state, tile2: !state.tile2 })} />
                <TileToggle checked={state.tile3} name={_(t.logging)} onChange={() => setState({ ...state, tile3: !state.tile3 })} />
            </TileToggleRow>
            <div style={{ paddingTop: 16 }}>
                <SimpleStripTable
                    rows={[
                        { attribute: _(t.id), value: deviceId },
                        { attribute: _(t.model), value: 'TODO: doorbell-2020-11-03' },
                        {
                            attribute: _(t.types),
                            value: types ? (
                                <TagGroup>
                                    {types.map((t, key) => (
                                        <Tag key={`${key}-${t}`}>{t}</Tag>
                                    ))}
                                </TagGroup>
                            ) : (
                                <div>-</div>
                            ),
                        },
                        {
                            attribute: _(t.firmware),
                            value: (
                                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                    <span style={{ marginRight: 6 }}>0.22.1</span> <Icon icon='cloud-success' size={24} />
                                </div>
                            ),
                        },
                        { attribute: _(t.status), value: 'TODO: 3 pending commands' },
                    ]}
                />
            </div>
        </div>
    )
}

export default Tab1
