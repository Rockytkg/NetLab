import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Radio,
  Result,
  Row,
  Select,
  Switch,
  Tabs,
  Typography,
  theme,
} from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import type { RadiusCertItem, RadiusEapMethod, RadiusEapSettings } from '@/types/radius'

/** EAP 表单值：enabledHandlers 拆成「全部/自定义」两个控件承载，提交时再拼回字符串。 */
interface EapFormValues {
  enabled: boolean
  method: RadiusEapMethod
  handlersMode: 'all' | 'custom'
  handlersCustom: RadiusEapMethod[]
  tlsServerCertId: number
  tlsClientCaId: number
  tlsMinVersion: '1.2' | '1.3'
}

const EAP_METHODS: RadiusEapMethod[] = ['eap-md5', 'eap-mschapv2', 'eap-tls', 'eap-peap', 'eap-ttls']

// 需要服务器证书的 EAP 方法
const CERT_REQUIRED_METHODS: RadiusEapMethod[] = ['eap-tls', 'eap-peap', 'eap-ttls']

const EAP_METHOD_LABELS: Record<RadiusEapMethod, string> = {
  'eap-md5': 'EAP-MD5',
  'eap-mschapv2': 'EAP-MSCHAPv2',
  'eap-tls': 'EAP-TLS',
  'eap-peap': 'PEAP',
  'eap-ttls': 'EAP-TTLS',
}

/** 802.1X 认证配置页：EAP 方法开关、启用的 Handler 与 TLS 证书绑定。 */
export default function RadiusDot1xPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [form] = Form.useForm<EapFormValues>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [serverCerts, setServerCerts] = useState<RadiusCertItem[]>([])
  const [caCerts, setCaCerts] = useState<RadiusCertItem[]>([])
  const handlersMode = Form.useWatch('handlersMode', form)
  const method = Form.useWatch('method', form)

  // enabledHandlers 字符串 ↔ 表单两控件
  const applyEapSettings = useCallback(
    (eap: RadiusEapSettings) => {
      const isAll = !eap.enabledHandlers || eap.enabledHandlers.trim() === '*'
      form.setFieldsValue({
        enabled: eap.enabled,
        method: eap.method,
        handlersMode: isAll ? 'all' : 'custom',
        handlersCustom: isAll
          ? [...EAP_METHODS]
          : (eap.enabledHandlers
              .split(',')
              .map((item) => item.trim())
              .filter((item): item is RadiusEapMethod =>
                EAP_METHODS.includes(item as RadiusEapMethod),
              )),
        tlsServerCertId: eap.tlsServerCertId,
        tlsClientCaId: eap.tlsClientCaId,
        tlsMinVersion: eap.tlsMinVersion,
      })
    },
    [form],
  )

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const [settings, servers, cas] = await Promise.all([
        radiusApi.getSettings(),
        radiusApi.listCerts({ size: 200, certType: 'server' }),
        radiusApi.listCerts({ size: 200, certType: 'ca' }),
      ])
      applyEapSettings(settings.eap)
      setServerCerts(servers.items ?? [])
      setCaCerts(cas.items ?? [])
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRadius, applyEapSettings])

  useEffect(() => {
    load()
  }, [load])

  const handleSave = async (values: EapFormValues) => {
    setSaving(true)
    try {
      const payload: RadiusEapSettings = {
        enabled: values.enabled,
        method: values.method,
        enabledHandlers:
          values.handlersMode === 'all' ? '*' : (values.handlersCustom ?? []).join(','),
        tlsServerCertId: values.tlsServerCertId ?? 0,
        tlsClientCaId: values.tlsClientCaId ?? 0,
        tlsMinVersion: values.tlsMinVersion,
      }
      const updated = await radiusApi.updateEapSettings(payload)
      applyEapSettings(updated)
      message.success(t('radius:common.saveSuccess'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  // 证书下拉：0 表示未选择
  const certOptions = (certs: RadiusCertItem[]) => [
    { value: 0, label: t('radius:settings.certNone') },
    ...certs.map((cert) => ({ value: cert.id, label: cert.name })),
  ]

  const methodLabel = (m: RadiusEapMethod) =>
    CERT_REQUIRED_METHODS.includes(m)
      ? `${EAP_METHOD_LABELS[m]} (${t('radius:eap.needsServerCert')})`
      : EAP_METHOD_LABELS[m]

  if (!canReadRadius) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div>
      <Card variant="outlined" loading={loading}>
        <Alert
          type="info"
          showIcon
          title={t('radius:eap.intro')}
          style={{ marginBottom: token.margin }}
        />
        <Form form={form} layout="vertical" requiredMark={false} onFinish={handleSave}>
          <Tabs
            tabBarStyle={{ marginBottom: token.marginLG }}
            items={[
              {
                key: 'general',
                label: t('radius:eap.sections.general'),
                children: (
                  <>
                    <Row gutter={[24, 0]}>
                      <Col xs={24} sm={8}>
                        <Form.Item name="enabled" label={t('radius:eap.form.enabled')} valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item
                      name="method"
                      label={t('radius:eap.form.method')}
                      tooltip={t('radius:eap.form.methodTip')}
                      rules={[{ required: true, message: t('radius:eap.form.methodRequired') }]}
                      extra={
                        method ? (
                          <Typography.Text type="secondary">
                            {t(`radius:eap.methodDesc.${method}`)}
                          </Typography.Text>
                        ) : undefined
                      }
                    >
                      <Radio.Group
                        optionType="button"
                        buttonStyle="solid"
                        options={EAP_METHODS.map((m) => ({ value: m, label: EAP_METHOD_LABELS[m] }))}
                      />
                    </Form.Item>
                    <Form.Item label={t('radius:eap.form.enabledHandlers')}>
                      <Form.Item name="handlersMode" noStyle>
                        <Radio.Group
                          options={[
                            { value: 'all', label: t('radius:eap.handlersAll') },
                            { value: 'custom', label: t('radius:eap.handlersCustom') },
                          ]}
                        />
                      </Form.Item>
                    </Form.Item>
                    {handlersMode === 'custom' && (
                      <Form.Item
                        name="handlersCustom"
                        rules={[
                          {
                            validator: (_, value?: RadiusEapMethod[]) =>
                              value && value.length > 0
                                ? Promise.resolve()
                                : Promise.reject(new Error(t('radius:eap.handlersRequired'))),
                          },
                        ]}
                      >
                        <Checkbox.Group
                          options={EAP_METHODS.map((m) => ({
                            value: m,
                            label: methodLabel(m),
                          }))}
                        />
                      </Form.Item>
                    )}
                  </>
                ),
              },
              {
                key: 'tls',
                label: t('radius:eap.sections.tls'),
                children: (
                  <>
                    <Row gutter={[24, 0]}>
                      <Col xs={24} sm={12}>
                        <Form.Item
                          name="tlsServerCertId"
                          label={t('radius:eap.form.tlsServerCertId')}
                          tooltip={t('radius:eap.form.tlsServerCertIdTip')}
                        >
                          <Select showSearch={{ optionFilterProp: 'label' }} options={certOptions(serverCerts)} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} sm={12}>
                        <Form.Item
                          name="tlsClientCaId"
                          label={t('radius:eap.form.tlsClientCaId')}
                          tooltip={t('radius:eap.form.tlsClientCaIdTip')}
                        >
                          <Select showSearch={{ optionFilterProp: 'label' }} options={certOptions(caCerts)} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={[24, 0]}>
                      <Col xs={24} sm={12}>
                        <Form.Item
                          name="tlsMinVersion"
                          label={t('radius:eap.form.tlsMinVersion')}
                          rules={[{ required: true, message: t('radius:eap.form.tlsMinVersionRequired') }]}
                        >
                          <Select
                            options={[
                              { value: '1.2', label: 'TLS 1.2' },
                              { value: '1.3', label: 'TLS 1.3' },
                            ]}
                          />
                        </Form.Item>
                      </Col>
                    </Row>
                  </>
                ),
              },
            ]}
          />

          <Can permission="radius.manage">
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
              {t('common:save')}
            </Button>
          </Can>
        </Form>
      </Card>
    </div>
  )
}
