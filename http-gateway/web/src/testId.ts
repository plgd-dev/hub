const testId = {
    app: {
        logout: 'hub-app-logout',
        layout: 'hub-app-layout',
    },
    menu: {
        apiTokens: 'hub-menu-item-api-tokens',
        devices: 'hub-menu-item-devices',
        snippetService: {
            link: 'hub-menu-item-snippet-service',
            configurations: 'hub-menu-item-snippet-service-configurations',
            conditions: 'hub-menu-item-snippet-service-conditions',
            appliedConfigurations: 'hub-menu-item-snippet-service-applied-configurations',
        },
    },
    devices: {
        detail: {
            layout: 'hub-devices-detail-layout',
            tabInformation: 'hub-devices-detail-tab-information',
            tabResources: 'hub-devices-detail-tab-resources',
            tabCertificates: 'hub-devices-detail-tab-certificates',
            tabProvisioningRecords: 'hub-devices-detail-tab-provisioning-records',
            informationTableId: 'hub-devices-detail-information-table-id',
            editNameButton: 'hub-devices-detail-edit-name-button',
            editNameModal: 'hub-devices-detail-edit-name-modal',
            deleteDeviceButton: 'hub-devices-detail-delete-device-button',
            deleteDeviceModal: 'hub-devices-detail-delete-device-modal',
            deleteDeviceButtonCancel: 'hub-devices-detail-delete-device-button-cancel',
            deleteDeviceButtonDelete: 'hub-devices-detail-delete-device-button-delete',
            information: {
                twinToggle: 'hub-devices-detail-information-twin-toggle',
                notificationsToggle: 'hub-devices-detail-information-notifications-toggle',
                endpoints: 'hub-devices-detail-information-endpoints',
                types: 'hub-devices-detail-information-types',
            },
            resources: {
                table: 'hub-devices-detail-resources-table',
                tree: 'hub-devices-detail-resources-tree',
                updateModal: 'hub-devices-detail-resources-update-modal',
                viewSwitch: 'hub-devices-detail-resources-view-switch',
                deleteModal: 'hub-devices-detail-resources-delete-modal',
            },
        },
    },
    remoteClients: {
        detail: {
            tabInformation: 'hub-remote-clients-detail-tab-information',
            tabConfiguration: 'hub-remote-clients-detail-tab-configuration',
        },
    },
    dps: {
        provisioningRecords: {
            detail: {
                deleteButton: 'hub-dps-provisioning-records-detail-delete-button',
                editNameButton: 'hub-dps-provisioning-records-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-provisioning-records-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-provisioning-records-detail-delete-button-confirm',
                editNameModal: 'hub-dps-provisioning-records-detail-edit-name-modal',
                tabDetails: 'hub-dps-provisioning-records-detail-tab-details',
                tabCredentials: 'hub-dps-provisioning-records-detail-tab-credentials',
                tabAcls: 'hub-dps-provisioning-records-detail-tab-acls',
            },
        },
        enrollmentGroups: {
            detail: {
                tabEnrollmentConfiguration: 'hub-dps-enrollment-groups-detail-tab-enrollment-configuration',
                tabDeviceCredentials: 'hub-dps-enrollment-groups-detail-tab-device-credentials',
                deleteButton: 'hub-dps-enrollment-groups-detail-delete-button',
                deleteButtonCancel: 'hub-dps-enrollment-groups-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-enrollment-groups-detail-delete-button-confirm',
                editNameButton: 'hub-dps-enrollment-groups-detail-edit-name-button',
                editNameModal: 'hub-dps-enrollment-groups-detail-edit-name-modal',
            },
        },
        linkedHubs: {
            detail: {
                tabDetails: 'hub-dps-linked-hubs-detail-tab-details',
                tabCertificateAuthority: 'hub-dps-linked-hubs-detail-tab-certificate-authority',
                tabAuthorization: 'hub-dps-linked-hubs-detail-tab-Authorization',
                deleteButton: 'hub-dps-linked-hubs-detail-delete-button',
                editNameButton: 'hub-dps-linked-hubs-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-linked-hubs-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-linked-hubs-detail-delete-button-confirm',
                editNameModal: 'hub-dps-linked-hubs-detail-edit-name-modal',
            },
        },
        certificates: {
            detail: {
                tabCertificateAuthorityConfiguration: 'hub-dps-certificates-detail-tab-certificate-authority-configuration',
                tabAuthorization: 'hub-dps-certificates-detail-tab-Authorization',
                deleteButton: 'hub-dps-certificates-detail-delete-button',
                editNameButton: 'hub-dps-certificates-detail-edit-name-button',
                deleteButtonCancel: 'hub-dps-certificates-detail-delete-button-cancel',
                deleteButtonConfirm: 'hub-dps-certificates-detail-delete-button-confirm',
                editNameModal: 'hub-dps-certificates-detail-edit-name-modal',
            },
        },
    },
    snippetService: {
        configurations: {
            addPage: {
                form: {
                    name: 'hub-snippet-service-configurations-add-page-form-name',
                    addResourceButton: 'hub-snippet-service-configurations-add-page-form-add-resource-button',
                    createResourceModal: 'hub-snippet-service-configurations-add-page-form-create-resource-modal',
                    resourceTable: 'hub-snippet-service-configurations-add-page-form-resource-table',
                    addButton: 'hub-snippet-service-configurations-add-page-form-add-button',
                    resetButton: 'hub-snippet-service-configurations-add-page-form-reset-button',
                },
            },
            list: {
                table: 'hub-snippet-service-configurations-list-table',
                addConfigurationButton: 'hub-snippet-service-configurations-list-add-configuration-button',
                invokeModal: 'hub-snippet-service-configurations-list-invoke-modal',
            },
            detail: {
                deleteButton: 'hub-snippet-service-configurations-detail-delete-button',
                deleteButtonConfirm: 'hub-snippet-service-configurations-detail-delete-button-confirm',
                deleteButtonCancel: 'hub-snippet-service-configurations-detail-delete-button-cancel',
                tabGeneral: 'hub-snippet-service-configurations-detail-tab-general',
                tabConditions: 'hub-snippet-service-configurations-detail-tab-conditions',
                tabAppliedConfiguration: 'hub-snippet-service-configurations-detail-tab-applied-configuration',
                versionSelector: 'hub-snippet-service-configurations-detail-version-selector',
                invokeButton: 'hub-snippet-service-configurations-detail-invoke-button',
                invokeModal: 'hub-snippet-service-configurations-detail-invoke-modal',
                deleteModal: 'hub-snippet-service-configurations-detail-delete-modal',
                saveButton: 'hub-snippet-service-configurations-detail-save-button',
                resetButton: 'hub-snippet-service-configurations-detail-reset-button',
                conditionsTable: 'hub-snippet-service-configurations-detail-conditions-table',
                appliedConfigurationsTable: 'hub-snippet-service-configurations-detail-applied-configurations-table',
            },
        },
        conditions: {
            list: {
                table: 'hub-snippet-service-conditions-list-table',
                addButton: 'hub-snippet-service-conditions-list-add-button',
                pageTemplate: 'hub-snippet-service-conditions-list-page-template',
            },
            detail: {
                deleteButton: 'hub-snippet-service-conditions-detail-delete-button',
                deleteButtonConfirm: 'hub-snippet-service-conditions-detail-delete-button-confirm',
                deleteButtonCancel: 'hub-snippet-service-conditions-detail-delete-button-cancel',
                tabGeneral: 'hub-snippet-service-conditions-detail-tab-general',
                tabFilters: 'hub-snippet-service-conditions-detail-tab-filters',
                tabApiAccessToken: 'hub-snippet-service-conditions-detail-tab-api-access-token',
                versionSelector: 'hub-snippet-service-conditions-detail-version-selector',
                deleteModal: 'hub-snippet-service-conditions-detail-delete-modal',
                tab1: {
                    form: {
                        name: 'hub-snippet-service-conditions-detail-tab1-form-name',
                    },
                },
                bottomPanel: 'hub-snippet-service-conditions-detail-bottom-panel',
                bottomPanelSave: 'hub-snippet-service-conditions-detail-bottom-panel-save',
                bottomPanelReset: 'hub-snippet-service-conditions-detail-bottom-panel-reset',
            },
            addPage: {
                wizard: 'hub-snippet-service-conditions-add-page-wizard',
                step1: {
                    form: {
                        name: 'hub-snippet-service-conditions-add-page-form-name',
                    },
                    buttons: 'hub-snippet-service-conditions-add-page-step1-buttons',
                },
                step2: {
                    filterDeviceId: 'hub-snippet-service-conditions-add-page-step2-filter-device-id',
                    selectDeviceId: 'hub-snippet-service-conditions-add-page-step2-select-device-id',
                    selectDeviceIdReset: 'hub-snippet-service-conditions-add-page-step2-select-device-id-reset',
                    selectDeviceIdDone: 'hub-snippet-service-conditions-add-page-step2-select-device-id-done',
                    resourceType: 'hub-snippet-service-conditions-add-page-step2-resource-type',
                    hrefFilter: 'hub-snippet-service-conditions-add-page-step2-href-filter',
                    jqExpressionFilter: 'hub-snippet-service-conditions-add-page-step2-jq-expression-filter',
                    buttons: 'hub-snippet-service-conditions-add-page-step2-buttons',
                },
                step3: {
                    selectConfiguration: 'hub-snippet-service-conditions-add-page-step3-select-configuration',
                    apiToken: 'hub-snippet-service-conditions-add-page-step3-api-token',
                    generateApiToken: 'hub-snippet-service-conditions-add-page-step3-generate-api-token',
                    generateApiTokenModal: 'hub-snippet-service-conditions-add-page-step3-generate-api-token-modal',
                    buttons: 'hub-snippet-service-conditions-add-page-step3-buttons',
                },
            },
        },
        appliedConfigurations: {
            detail: {
                deleteButton: 'hub-snippet-service-applied-configurations-detail-delete-button',
                deleteButtonConfirm: 'hub-snippet-service-applied-configurations-detail-delete-button-confirm',
                deleteButtonCancel: 'hub-snippet-service-applied-configurations-detail-delete-button-cancel',
                tabGeneral: 'hub-snippet-service-applied-configurations-detail-tab-general',
                tabListOfResources: 'hub-snippet-service-applied-configurations-detail-tab-list-of-resources',
                deleteModal: 'hub-snippet-service-applied-configurations-detail-delete-modal',
                tab1: {
                    configurationLink: 'hub-snippet-service-applied-configurations-detail-tab1-configuration-link',
                    conditionLink: 'hub-snippet-service-applied-configurations-detail-tab1-condition-link',
                },
                tab2: {
                    resourceToggleCreator: 'hub-snippet-service-applied-configurations-detail-resource-toggle-creator',
                },
            },
            list: {
                pageTemplate: 'hub-snippet-service-applied-configurations-list-page-template',
                table: 'hub-snippet-service-applied-configurations-list-table',
            },
        },
    },
    apiTokens: {
        detail: {
            deleteButton: 'hub-snippet-service-api-tokens-detail-delete-button',
            deleteButtonCancel: 'hub-snippet-service-api-tokens-detail-delete-button-cancel',
            deleteButtonConfirm: 'hub-snippet-service-api-tokens-detail-delete-button-confirm',
            deleteModal: 'hub-snippet-service-api-tokens-detail-delete-modal',
            simpleTableLeft: 'hub-snippet-service-api-tokens-detail-simple-table-left',
            simpleTableRight: 'hub-snippet-service-api-tokens-detail-simple-table-right',
            tableGlobalFilter: 'hub-snippet-service-api-tokens-detail-table-global-filter',
        },
        list: {
            table: 'hub-snippet-service-api-tokens-list-table',
            createTokenButton: 'hub-snippet-service-api-tokens-list-create-token-button',
            addModal: 'hub-snippet-service-api-tokens-list-add-modal',
            page: 'hub-snippet-service-api-tokens-list-page',
        },
    },
}

export default testId
