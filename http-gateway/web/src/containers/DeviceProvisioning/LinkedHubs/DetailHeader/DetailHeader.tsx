import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import { DeleteModal, IconTrash } from '@shared-ui/components/Atomic'

import { Props } from './DetailHeader.types'
import * as styles from './DetailHeader.styles'
import testId from '@/testId'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import { deleteLinkedHubsApi } from '@/containers/DeviceProvisioning/rest'

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { id, loading, refresh } = props

    const [deleteModal, setDeleteModal] = useState(false)
    const [deleting, setDeleting] = useState(false)

    const handleDelete = useCallback(async () => {
        try {
            if (id) {
                setDeleting(true)

                await deleteLinkedHubsApi([id])

                setDeleting(false)
                setDeleteModal(false)
                refresh()
            }
        } catch (e: any) {
            setDeleting(false)
            setDeleteModal(false)
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
