import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import cloneDeep from 'lodash/cloneDeep'
import { Controller } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import CaPool, { CA_BASE64_PREFIX } from '@shared-ui/components/Organisms/CaPool'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import Modal from '@shared-ui/components/Atomic/Modal'
import CodeEditor from '@shared-ui/components/Atomic/CodeEditor'
import { FormContext } from '@shared-ui/common/context/FormContext'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { parse, pemToDER } from '@shared-ui/common/utils/cert-decoder.mjs'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { formatCertName, parseCertificate } from '@shared-ui/common/services/certificates'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import * as styles from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/Tabs/Tab.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import notificationId from '@/notificationId'
import { stringToPem } from '../../utils'

const modalVariants = {
    ADD_CA_POOL: 'addCaPool',
    EDIT_PRIVATE_KEY: 'editPrivateKey',
    EDIT_CERT: 'editCert',
}

type ModalData = {
    title: string
    description?: string
    value: string
    variant: (typeof modalVariants)[keyof typeof modalVariants]
}

const TlsPage: FC<any> = (props) => {
    const { contentRefs, control, loading, prefix, setValue, watch } = props

    const { formatMessage: _ } = useIntl()
    const { i18n } = useContext(FormContext)

    const [caPoolData, setCaPoolData] = useState<any>([])
    const [caPoolLoading, setCaPoolLoading] = useState(false)
    const [certData, setCertData] = useState<any>([])
    const [certLoading, setCertLoading] = useState<any>(false)

    const caPool = watch(`${prefix}tls.caPool`)
    const key = watch(`${prefix}tls.key`)
    const cert = watch(`${prefix}tls.cert`)

    useEffect(() => {
        const parseCaPool = async (certs: any, singleMode = false) => {
            const parsed = certs?.map(async (p: string, key: number) => {
                try {
                    if (p.startsWith('/')) {
                        return { id: key, name: p, data: undefined }
                    }

                    const certsData = atob(p.replace(CA_BASE64_PREFIX, ''))

                    if (singleMode) {
                        const data = await parse(pemToDER(certsData.replace(/(-----(BEGIN|END) CERTIFICATE-----|[\n\r])/g, ''))).then((c) => c)
                        return { id: key, name: formatCertName(data), data: data, dataChain: p }
                    } else {
                        return await parseCertificate(certsData, key)
                    }
                } catch (e: any) {
                    Notification.error(
                        { title: _(t.certificationParsingError), message: getApiErrorMessage(e) },
                        { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_CERT_PARSE_ERROR }
                    )
                }
            })

            return await Promise.all(parsed)
        }

        if (caPool) {
            setCaPoolLoading(true)
            parseCaPool(caPool)
                .catch(console.error)
                .then((c) => {
                    setCaPoolLoading(false)
                    setCaPoolData(c)
                })
        }

        if (cert) {
            setCertLoading(true)
            parseCaPool([cert], true)
                .catch(console.error)
                .then((c) => {
                    setCertLoading(false)
                    setCertData(c)
                })
        }

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [caPool, cert])

    const defaultModalData = useMemo(
        () => ({
            title: '',
            value: '',
            description: undefined,
            variant: modalVariants.ADD_CA_POOL,
        }),
        []
    )

    const [modalData, setModalData] = useState<ModalData>(defaultModalData)
    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: _(t.caPoolDetail),
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const caModalDataValid = useMemo(
        () => modalData.value === undefined || modalData.value.startsWith('/') || modalData.value.startsWith('-----BEGIN'),
        [modalData.value]
    )

    const handleSaveModalData = useCallback(() => {
        switch (modalData.variant) {
            case modalVariants.ADD_CA_POOL: {
                setValue(`${prefix}tls.caPool`, [...caPool, stringToPem(modalData.value)], {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                break
            }
            case modalVariants.EDIT_PRIVATE_KEY: {
                setValue(`${prefix}tls.key`, stringToPem(modalData.value), {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                break
            }
            case modalVariants.EDIT_CERT: {
                setValue(`${prefix}tls.cert`, stringToPem(modalData.value), {
                    shouldDirty: true,
                    shouldTouch: true,
                })
                break
            }
        }

        setModalData(defaultModalData)
    }, [caPool, defaultModalData, modalData.value, modalData.variant, prefix, setValue])

    const handleDeleteCaItem = useCallback(
        (id: string) => {
            const newData = cloneDeep(caPool)
            newData.splice(parseInt(id, 10), 1)
            setValue(`${prefix}tls.caPool`, newData, { shouldDirty: true, shouldTouch: true })
        },
        [caPool, prefix, setValue]
    )

    const handleDownload = useCallback(
        (id: string) => {
            const caItem = caPoolData.find((item: { id: string; name: string; data: {}[] }) => item.id === id)

            const link = document.createElement('a')
            document.body.appendChild(link)
            link.href = window.URL.createObjectURL(
                new Blob([atob(caItem.dataChain.replace(CA_BASE64_PREFIX, ''))], {
                    type: 'application/x-pem-file',
                })
            )
            link.setAttribute('download', 'PEM(chain).pem')
            link.click()
            document.body.removeChild(link)
        },
        [caPoolData]
    )

    const handleViewCa = useCallback(
        (id: string) => {
            const caItem = caPoolData.find((item: { id: string; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.caPoolDetail), subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [caPoolData]
    )

    const handleViewCert = useCallback(
        (id: string) => {
            const certItem = certData.find((item: { id: string; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.certificateDetail), subTitle: certItem.name, data: [certItem.data] || certItem.name, dataChain: certItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    const commonI18n = useMemo(
        () => ({
            title: _(g.uploadedCaPools),
            download: _(g.download),
            edit: _(g.edit),
            delete: _(g.delete),
            search: _(g.search),
            showMore: _(g.showMore),
            update: _(g.update),
            view: _(g.view),
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <form>
            <Headline type='h5'>{_(g.tls)}</Headline>
            <p>{_(g.tlsDescription)}</p>
            <hr css={styles.separator} />
            <Loadable condition={!caPoolLoading}>
                <CaPool
                    customComponent={
                        <Loadable condition={!loading}>
                            <Controller
                                control={control}
                                name='certificateAuthority.grpc.tls.useSystemCaPool'
                                render={({ field: { onChange, value } }) => (
                                    <TileToggle checked={value ?? false} name={_(t.useSystemCAPool)} onChange={onChange} />
                                )}
                            />
                        </Loadable>
                    }
                    data={caPoolData}
                    headline={_(t.caPool)}
                    headlineRef={contentRefs.ref1}
                    i18n={commonI18n}
                    onAdd={() =>
                        setModalData({
                            title: _(t.addCaPool),
                            description: undefined,
                            value: '',
                            variant: modalVariants.ADD_CA_POOL,
                        })
                    }
                    onDelete={handleDeleteCaItem}
                    onDownload={handleDownload}
                    onView={handleViewCa}
                    tableSearch={true}
                />
            </Loadable>
            <Spacer type='pt-8'>
                <Loadable condition={!loading}>
                    <CaPool
                        data={key ? [{ id: '0', name: key }] : []}
                        headline={_(t.privateKey)}
                        headlineRef={contentRefs.ref2}
                        i18n={commonI18n}
                        onAdd={
                            key
                                ? undefined
                                : () =>
                                      setModalData({
                                          title: _(t.addPrivateKey),
                                          description: undefined,
                                          value: '',
                                          variant: modalVariants.EDIT_PRIVATE_KEY,
                                      })
                        }
                        onDelete={() => setValue(`${prefix}tls.key`, '', { shouldDirty: true, shouldTouch: true })}
                        onEdit={() =>
                            setModalData({
                                title: _(t.editPrivateKey),
                                description: undefined,
                                value: key.startsWith('/') ? key : atob(key.replace(CA_BASE64_PREFIX, '')),
                                variant: modalVariants.EDIT_PRIVATE_KEY,
                            })
                        }
                    />
                </Loadable>
            </Spacer>
            <Spacer type='pt-8'>
                <Loadable condition={!loading && !certLoading}>
                    <CaPool
                        data={cert ? certData : []}
                        headline={_(t.certificate)}
                        headlineRef={contentRefs.ref3}
                        i18n={commonI18n}
                        onAdd={
                            cert
                                ? undefined
                                : () =>
                                      setModalData({
                                          title: _(t.addCertificate),
                                          description: undefined,
                                          value: '',
                                          variant: modalVariants.EDIT_CERT,
                                      })
                        }
                        onDelete={() => setValue(`${prefix}tls.cert`, '', { shouldDirty: true, shouldTouch: true })}
                        onEdit={
                            cert && (cert?.startsWith('/') || certData[0])
                                ? () =>
                                      setModalData({
                                          title: _(t.editCertificate),
                                          description: undefined,
                                          value: key.startsWith('/') ? cert : certData[0].data.files.pem.replace(/%0D%0A/g, '\n').replace(/%20/g, ' '),
                                          variant: modalVariants.EDIT_CERT,
                                      })
                                : undefined
                        }
                        onView={cert?.startsWith('/') ? undefined : handleViewCert}
                    />
                </Loadable>
            </Spacer>
            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
            <Modal
                appRoot={document.getElementById('root')}
                bodyStyle={{
                    overflow: 'hidden',
                }}
                footerActions={[
                    {
                        label: _(g.resetChanges),
                        onClick: () => setModalData(defaultModalData),
                        variant: 'tertiary',
                    },
                    {
                        label: _(g.saveChanges),
                        onClick: handleSaveModalData,
                        variant: 'primary',
                        disabled: !caModalDataValid,
                    },
                ]}
                maxWidth={1100}
                onClose={() => setModalData(defaultModalData)}
                portalTarget={document.getElementById('modal-root')}
                renderBody={
                    <CodeEditor
                        onChange={(v) => setModalData((p) => ({ ...p, value: v }))}
                        placeholderText={_(g.editorPlaceholder)}
                        value={modalData?.value || ''}
                    />
                }
                show={modalData.title !== ''}
                title={modalData?.title}
                width='100%'
                zIndex={25}
            />
        </form>
    )
}

TlsPage.displayName = 'TlsPage'

export default TlsPage
