import React, { FC, useState } from 'react'

import TileToggleRow from '@shared-ui/components/Atomic/TileToggle/TileToggleRow'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'

const Tab2: FC<any> = () => {
    const [switchers, setSwitchers] = useState({
        switch1: false,
        switch2: true,
    })
    return (
        <div>
            <TileToggleRow>
                <TileToggle checked={switchers.switch1} name='Switcher 1' onChange={() => setSwitchers((prev) => ({ ...switchers, switch1: !prev.switch1 }))} />
                <TileToggle checked={switchers.switch2} name='Switcher 2' onChange={() => setSwitchers((prev) => ({ ...switchers, switch2: !prev.switch2 }))} />
            </TileToggleRow>
            <Spacer type='pt-4'>
                <SimpleStripTable
                    rows={[
                        { attribute: 'Attribute 1', value: 'value1' },
                        { attribute: 'Attribute 2', value: 'value2' },
                        { attribute: 'Attribute 3', value: 'value3' },
                        { attribute: 'Attribute 4', value: 'value4' },
                        { attribute: 'Attribute 5', value: 'value5' },
                    ]}
                />
            </Spacer>
        </div>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
