import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import { DeleteModal, IconEdit, IconTrash } from '@shared-ui/components/Atomic'
import EditNameModal from '@shared-ui/components/Organisms/EditNameModal'

import { Props } from './DetailHeader.types'
import * as styles from './DetailHeader.styles'
import testId from '@/testId'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'

const DetailHeader: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { id, loading, refresh } = props

    const [deleteModal, setDeleteModal] = useState(false)
    const [deleting, setDeleting] = useState(false)
    const [editNameModal, setEditNameModal] = useState(false)
    const [editing, setEditing] = useState(false)

    const handleDelete = useCallback(async () => {
        try {
            if (id) {
                setDeleting(true)

                // DELETE

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

    const handleUpdateName = useCallback(
        async (name: string) => {
            try {
                setEditing(true)

                // UPDATE
                console.log(name)

                setEditing(false)
                setEditNameModal(false)
                refresh()
            } catch (e: any) {
                setEditing(false)
                setEditNameModal(false)
            }
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <div css={styles.list}>
            <Button
                dataTestId={testId.dps.enrollmentGroups.detail.deleteButton}
                disabled={loading}
                icon={<IconTrash />}
                onClick={() => setDeleteModal(true)}
                variant='tertiary'
            >
                {_(g.delete)}
            </Button>

            <Button
                dataTestId={testId.dps.enrollmentGroups.detail.editNameButton}
                disabled={loading}
                icon={<IconEdit />}
                onClick={() => setEditNameModal(true)}
                style={{ marginLeft: 8 }}
                variant='tertiary'
            >
                {_(g.editName)}
            </Button>

            <DeleteModal
                deleteInformation={[
                    { label: _(g.name), value: 'TODO' },
                    { label: _(g.id), value: id },
                ]}
                footerActions={[
                    {
                        dataTestId: testId.dps.enrollmentGroups.detail.deleteButtonCancel,
                        label: _(g.cancel),
                        onClick: () => setDeleteModal(false),
                        variant: 'tertiary',
                    },
                    {
                        dataTestId: testId.dps.enrollmentGroups.detail.deleteButtonConfirm,
                        label: _(g.delete),
                        loading: deleting,
                        loadingText: _(g.deleting),
                        onClick: handleDelete,
                        variant: 'primary',
                    },
                ]}
                onClose={() => setDeleteModal(false)}
                show={deleteModal}
                subTitle={_(t.deleteEnrollmentGroupTitle)}
                title={_(t.deleteEnrollmentGroupMessage)}
            />

            <EditNameModal
                dataTestId={testId.dps.enrollmentGroups.detail.editNameModal}
                handleClose={() => setEditNameModal(false)}
                handleSubmit={handleUpdateName}
                i18n={{
                    close: _(g.close),
                    namePlaceholder: 'TODO',
                    edit: _(g.edit),
                    name: _(g.name),
                    reset: _(g.reset),
                    saveChange: _(g.saveChange),
                    savingChanges: _(g.savingChanges),
                }}
                loading={editing}
                name={id}
                show={editNameModal}
            />
        </div>
    )
}

DetailHeader.displayName = 'DetailHeader'

export default DetailHeader
