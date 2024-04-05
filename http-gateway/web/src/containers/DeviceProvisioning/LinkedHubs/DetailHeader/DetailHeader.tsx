import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { DeleteModal, IconTrash } from '@shared-ui/components/Atomic'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { Props } from './DetailHeader.types'
import * as styles from './DetailHeader.styles'
import testId from '@/testId'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import { deleteLinkedHubsApi } from '@/containers/DeviceProvisioning/rest'
import { pages } from '@/routes'
import notificationId from '@/notificationId'

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { id, loading } = props

    const navigate = useNavigate()

    const [deleteModal, setDeleteModal] = useState(false)
    const [deleting, setDeleting] = useState(false)

    const handleDelete = useCallback(async () => {
        try {
            if (id) {
                setDeleting(true)

                await deleteLinkedHubsApi([id])

                setDeleting(false)
                setDeleteModal(false)

                Notification.success(
                    { title: _(t.linkedHubDeleted), message: _(t.linkedHubDeletedMessage) },
                    { notificationId: notificationId.HUB_DPS_LINKED_HUBS_UPDATED }
                )

                navigate(generatePath(pages.DPS.LINKED_HUBS.LINK))
            }
        } catch (e: any) {
            setDeleting(false)
            setDeleteModal(false)

            Notification.error(
                { title: _(t.linkedHubsError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DELETED_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id])

    return (
        <div css={styles.list}>
            <Button
                dataTestId={testId.dps.linkedHubs.detail.deleteButton}
                disabled={loading}
                htmlType='button'
                icon={<IconTrash />}
                onClick={() => setDeleteModal(true)}
                variant='tertiary'
            >
                {_(g.delete)}
            </Button>

            <DeleteModal
                deleteInformation={[
                    { label: _(g.name), value: 'TODO' },
                    { label: _(g.id), value: id },
                ]}
                footerActions={[
                    {
                        dataTestId: testId.dps.linkedHubs.detail.deleteButtonCancel,
                        label: _(g.cancel),
                        onClick: () => setDeleteModal(false),
                        variant: 'tertiary',
                    },
                    {
                        dataTestId: testId.dps.linkedHubs.detail.deleteButtonConfirm,
                        label: _(g.delete),
                        loading: deleting,
                        loadingText: _(g.deleting),
                        onClick: handleDelete,
                        variant: 'primary',
                    },
                ]}
                onClose={() => setDeleteModal(false)}
                show={deleteModal}
                subTitle={_(t.deleteLinkedHubsTitle)}
                title={_(t.deleteLinkedHubMessage)}
            />
        </div>
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
