import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import cloneDeep from 'lodash/cloneDeep'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import CaList from '@shared-ui/components/Organisms/CaList/CaList'
import { useCaData } from '@shared-ui/common/hooks/useCaData'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import { Props } from './SubStepTls.types'
import { nameLengthValidator, stringToPem } from '@/containers/DeviceProvisioning/utils'

const SubStepTls: FC<Props> = (props) => {
    const { control, setValue, updateField, watch, prefix } = props

    const { formatMessage: _ } = useIntl()
    const i18n = useCaI18n()

    const caPool = watch(`${prefix}tls.caPool`)
    const key = watch(`${prefix}tls.key`)
    const cert = watch(`${prefix}tls.cert`)
    const useSystemCaPool = watch(`${prefix}tls.useSystemCaPool`)

    const { parsedData: caPoolData } = useCaData({
        data: caPool,
        onError: (e) =>
            Notification.error(
                { title: _(t.certificationParsingError), message: e },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR }
            ),
    })

    const { parsedData: certData } = useCaData({
        data: cert !== '' ? [cert] : [],
        onError: (e) =>
            Notification.error(
                { title: _(t.certificationParsingError), message: e },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR }
            ),
    })

    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: _(t.caPoolDetail),
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const handleDeleteCaItem = useCallback(
        (id: number) => {
            const newData = cloneDeep(caPool)
            newData.splice(id, 1)
            setValue(`${prefix}tls.caPool`, newData, { shouldDirty: true, shouldTouch: true })
        },
        [caPool, prefix, setValue]
    )

    const handleViewCa = useCallback(
        (id: number) => {
            const caItem = caPoolData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.caPoolDetail), subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [caPoolData]
    )

    const handleViewCert = useCallback(
        (id: number) => {
            const certItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.caPoolDetail), subTitle: certItem.name, data: certItem.data || certItem.name, dataChain: certItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    return (
        <div>
            <Spacer type='pt-12'>
                <h2 css={commonStyles.subHeadline}>TLS</h2>
            </Spacer>

            <FullPageWizard.Description>{_(t.tlsDescription)}</FullPageWizard.Description>

            <h3 css={commonStyles.groupHeadline}>{_(t.caPool)}</h3>

            <Spacer type='mb-5'>
                <Controller
                    control={control}
                    name={`${prefix}tls.useSystemCaPool`}
                    render={({ field: { onChange, value } }) => (
                        <TileToggle
                            darkBg
                            checked={(value as boolean) ?? false}
                            name={_(t.useSystemCAPool)}
                            onChange={(e) => {
                                updateField(`${prefix}tls.useSystemCaPool`, e.target.value === 'on')
                                onChange(e)
                            }}
                        />
                    )}
                />
            </Spacer>

            <FormLabel required={!useSystemCaPool} text={_(t.caPoolList)} tooltipText={_(t.requiredWithoutCaPool)} />
            <Dropzone
                smallPadding
                customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                description={_(t.uploadCertDescription)}
                maxFiles={10}
                onFilesDrop={(files) => {
                    setTimeout(() => {
                        setValue(`${prefix}tls.caPool`, [...caPool, ...files.map((f) => stringToPem(f))], {
                            shouldDirty: true,
                            shouldTouch: true,
                        })
                    }, 100)
                }}
                renderThumbs={false}
                title={_(t.uploadCertTitle)}
                validator={(file) => nameLengthValidator(file)}
            />

            {caPoolData && caPoolData.length > 0 && (
                <Spacer type='pt-6'>
                    <CaList
                        actions={{
                            onDelete: handleDeleteCaItem,
                            onView: handleViewCa,
                        }}
                        data={caPoolData}
                        i18n={{
                            title: _(g.uploadedCaPools),
                            download: _(g.download),
                            delete: _(g.delete),
                            view: _(g.view),
                        }}
                    />
                </Spacer>
            )}

            <Spacer type='pt-8'>
                <h3 css={commonStyles.groupHeadline}>{_(t.certificateKeyPair)}</h3>
            </Spacer>

            <FormGroup id={`${prefix}tls.key`}>
                <FormLabel marginBottom={key === ''} required={cert !== '' || key !== ''} text={_(t.key)} />

                {key === '' ? (
                    <Dropzone
                        smallPadding
                        customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                        description={_(t.uploadCertKeyDescription)}
                        maxFiles={1}
                        onFilesDrop={(files) => {
                            setTimeout(() => {
                                setValue(`${prefix}tls.key`, stringToPem(files[0]), {
                                    shouldDirty: true,
                                    shouldTouch: true,
                                })
                            }, 100)
                        }}
                        renderThumbs={false}
                        title={_(t.uploadCertTitle)}
                        validator={(file) => nameLengthValidator(file, true)}
                    />
                ) : (
                    <Spacer type='pt-2'>
                        <CaList
                            actions={{
                                onDelete: () => setValue(`${prefix}tls.key`, '', { shouldDirty: true, shouldTouch: true }),
                            }}
                            data={[{ id: 0, name: key, data: [], dataChain: '' }]}
                            i18n={{
                                title: '',
                                download: _(g.download),
                                delete: _(g.delete),
                                view: _(g.view),
                            }}
                        />
                    </Spacer>
                )}
            </FormGroup>

            <FormGroup id={`${prefix}tls.cert`}>
                <FormLabel marginBottom={cert === ''} required={cert !== '' || key !== ''} text={_(t.certificate)} />

                {cert === '' ? (
                    <Dropzone
                        smallPadding
                        customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                        description={_(t.uploadCertDescription)}
                        maxFiles={1}
                        onFilesDrop={(files) => {
                            setTimeout(() => {
                                setValue(`${prefix}tls.cert`, stringToPem(files[0]), {
                                    shouldDirty: true,
                                    shouldTouch: true,
                                })
                            }, 100)
                        }}
                        renderThumbs={false}
                        title={_(t.uploadCertTitle)}
                        validator={(file) => nameLengthValidator(file)}
                    />
                ) : (
                    <Spacer type='pt-2'>
                        <Loadable condition={certData.length > 0}>
                            <CaList
                                actions={{
                                    onDelete: () => setValue(`${prefix}tls.cert`, '', { shouldDirty: true, shouldTouch: true }),
                                    onView: handleViewCert,
                                }}
                                data={certData}
                                i18n={{
                                    title: '',
                                    download: _(g.download),
                                    delete: _(g.delete),
                                    view: _(g.view),
                                }}
                            />
                        </Loadable>
                    </Spacer>
                )}
            </FormGroup>

            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
        </div>
    )
}

SubStepTls.displayName = 'SubStepTls'

export default SubStepTls
