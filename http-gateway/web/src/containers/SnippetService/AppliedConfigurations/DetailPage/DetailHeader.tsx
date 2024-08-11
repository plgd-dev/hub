import { FC } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { DeleteInformationType } from '@shared-ui/components/Atomic/Modal/components/DeleteModal/DeleteModal.types'

import { messages as g } from '@/containers/Global.i18n'
import DetailHeaderLayout from '@/containers/Common/DetailHeaderLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { deleteAppliedConfigurationApi } from '@/containers/SnippetService/rest'
import testId from '@/testId'
import notificationId from '@/notificationId'
import { pages } from '@/routes'

type Props = {
    id: string
    configurationId: string
    configurationName?: string
    conditionId?: string
    conditionName?: string | number
    loading: boolean
    name?: string
}

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const navigate = useNavigate()

    return (
        <DetailHeaderLayout
            deleteApiMethod={deleteAppliedConfigurationApi}
            deleteInformation={
                [
                    { label: _(g.id), value: props.id },
                    { label: _(confT.configurationId), value: props.configurationId },
                    props.configurationName ? { label: _(confT.condition), value: props.configurationName } : undefined,
                    props.conditionId ? { label: _(confT.conditionId), value: props.conditionId } : undefined,
                    props.conditionName ? { label: _(confT.condition), value: props.conditionName } : undefined,
                ].filter((i) => !!i) as DeleteInformationType[]
            }
            i18n={{
                id: _(g.id),
                name: _(g.name),
                cancel: _(g.cancel),
                delete: _(g.delete),
                deleting: _(g.deleting),
                subTitle: _(confT.deleteAppliedConfigurationMessage),
                title: _(confT.deleteAppliedConfiguration),
            }}
            id={props.id}
            loading={props.loading}
            onDeleteError={(error: any) => {
                Notification.error(
                    { title: _(confT.appliedConfigurationError), message: getApiErrorMessage(error) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_DETAIL_PAGE_DELETE_ERROR }
                )
            }}
            onDeleteSuccess={() => {
                Notification.success(
                    { title: _(confT.appliedConfigurationDeleted), message: _(confT.appliedConfigurationDeletedMessage) },
                    { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_DETAIL_PAGE_DELETE_SUCCESS }
                )

                navigate(generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.LINK))
            }}
            testIds={{
                deleteButton: testId.snippetService.appliedConfigurations.detail.deleteButton,
                deleteButtonCancel: testId.snippetService.appliedConfigurations.detail.deleteButtonCancel,
                deleteButtonConfirm: testId.snippetService.appliedConfigurations.detail.deleteButtonConfirm,
                deleteModal: testId.snippetService.appliedConfigurations.detail.deleteModal,
            }}
        />
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
