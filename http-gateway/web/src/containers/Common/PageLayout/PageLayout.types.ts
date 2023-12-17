import { Props as PageLayoutProps } from '@shared-ui/components/Atomic/PageLayout/PageLayout.types'
import { BreadcrumbItem } from '@shared-ui/components/Layout/Header/Breadcrumbs/Breadcrumbs.types'

export type Props = PageLayoutProps & {
    breadcrumbs: BreadcrumbItem[]
    deviceId?: string
}
