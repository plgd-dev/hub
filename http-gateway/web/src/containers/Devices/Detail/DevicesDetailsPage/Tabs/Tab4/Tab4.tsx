import { FC, useCallback } from 'react'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'

import { Props } from './Tab4.types'
import { pages } from '@/routes'
import DetailPage from '@/containers/DeviceProvisioning/ProvisioningRecords/DetailPage/DetailPage'

const Tab4: FC<Props> = (props) => {
    const { provisioningRecords } = props

    const { formatMessage: _ } = useIntl()
    const { section, id } = useParams()
    const navigate = useNavigate()

    const handleItemClick = useCallback(
        (item: ItemType) => {
            navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[3], section: item.link }))
        },
        [id, navigate]
    )

    return <DetailPage currentTab={section} onItemClick={handleItemClick} provisioningRecord={provisioningRecords} />
}

Tab4.displayName = 'Tab4'

export default Tab4
