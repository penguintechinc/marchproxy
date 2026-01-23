{{- define "marchproxy-egress.podTemplate" -}}
metadata:
  annotations:
    checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
    {{- if .Values.envoyConfig.create }}
    checksum/envoy: {{ include (print $.Template.BasePath "/envoy-configmap.yaml") . | sha256sum }}
    {{- end }}
    {{- if .Values.tls.createSecret }}
    checksum/tls: {{ include (print $.Template.BasePath "/tls-secret.yaml") . | sha256sum }}
    {{- end }}
    {{- with .Values.podAnnotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  labels:
    {{- include "marchproxy-egress.selectorLabels" . | nindent 4 }}
    {{- with .Values.podLabels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  {{- with .Values.global.imagePullSecrets }}
  imagePullSecrets:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  serviceAccountName: {{ include "marchproxy-egress.serviceAccountName" . }}
  {{- with .Values.priorityClassName }}
  priorityClassName: {{ . }}
  {{- end }}
  securityContext:
    {{- toYaml .Values.securityContext.pod | nindent 4 }}
  containers:
    - name: egress
      image: {{ include "marchproxy-egress.image" . }}
      imagePullPolicy: {{ .Values.image.pullPolicy }}
      ports:
        - name: tcp-proxy
          containerPort: {{ .Values.service.tcpProxyTargetPort }}
          protocol: TCP
        - name: admin
          containerPort: {{ .Values.service.adminTargetPort }}
          protocol: TCP
        - name: http
          containerPort: {{ .Values.service.httpTargetPort }}
          protocol: TCP
        - name: https
          containerPort: {{ .Values.service.httpsTargetPort }}
          protocol: TCP
        - name: envoy-admin
          containerPort: {{ .Values.service.envoyAdminTargetPort }}
          protocol: TCP
        - name: ext-auth
          containerPort: {{ .Values.service.extAuthTargetPort }}
          protocol: TCP
      env:
        - name: PROXY_TYPE
          value: {{ .Values.config.proxyType | quote }}
        - name: PROXY_CONFIG_PATH
          value: {{ .Values.config.configPath | quote }}
        - name: PROXY_LOG_PATH
          value: {{ .Values.config.logPath | quote }}
        - name: PROXY_CERT_PATH
          value: {{ .Values.config.certPath | quote }}
        - name: LOG_LEVEL
          value: {{ .Values.config.logLevel | quote }}
        - name: MTLS_ENABLED
          value: {{ .Values.config.mtls.enabled | quote }}
        {{- if .Values.config.mtls.enabled }}
        - name: MTLS_SERVER_CERT_PATH
          value: {{ .Values.config.mtls.serverCertPath | quote }}
        - name: MTLS_SERVER_KEY_PATH
          value: {{ .Values.config.mtls.serverKeyPath | quote }}
        - name: MTLS_CLIENT_CA_PATH
          value: {{ .Values.config.mtls.clientCaPath | quote }}
        - name: MTLS_VERIFY_CLIENT
          value: {{ .Values.config.mtls.verifyClient | quote }}
        {{- end }}
        - name: L7_ENABLED
          value: {{ .Values.config.l7.enabled | quote }}
        {{- if .Values.config.l7.enabled }}
        - name: ENVOY_BINARY
          value: {{ .Values.config.l7.envoyBinary | quote }}
        - name: ENVOY_CONFIG_PATH
          value: {{ .Values.config.l7.envoyConfigPath | quote }}
        - name: ENVOY_ADMIN_PORT
          value: {{ .Values.config.l7.envoyAdminPort | quote }}
        - name: ENVOY_LISTEN_PORT
          value: {{ .Values.config.l7.envoyListenPort | quote }}
        - name: ENVOY_HTTPS_PORT
          value: {{ .Values.config.l7.envoyHttpsPort | quote }}
        - name: ENVOY_LOG_LEVEL
          value: {{ .Values.config.l7.envoyLogLevel | quote }}
        - name: ENVOY_HTTP3_ENABLED
          value: {{ .Values.config.l7.http3Enabled | quote }}
        {{- end }}
        - name: ENABLE_EBPF
          value: {{ .Values.config.ebpf.enabled | quote }}
        - name: ENABLE_XDP
          value: {{ .Values.config.xdp.enabled | quote }}
        {{- if .Values.config.xdp.enabled }}
        - name: XDP_INTERFACE
          value: {{ .Values.config.xdp.interface | quote }}
        {{- end }}
        - name: THREAT_IP_BLOCKING_ENABLED
          value: {{ .Values.config.threatIntel.ipBlockingEnabled | quote }}
        - name: THREAT_DOMAIN_BLOCKING_ENABLED
          value: {{ .Values.config.threatIntel.domainBlockingEnabled | quote }}
        - name: THREAT_URL_MATCHING_ENABLED
          value: {{ .Values.config.threatIntel.urlMatchingEnabled | quote }}
        - name: THREAT_DNS_CACHE_ENABLED
          value: {{ .Values.config.threatIntel.dnsCacheEnabled | quote }}
        - name: EXTAUTH_PORT
          value: {{ .Values.config.extAuth.port | quote }}
        - name: RATE_LIMIT_ENABLED
          value: {{ .Values.config.rateLimit.enabled | quote }}
        {{- if .Values.config.rateLimit.enabled }}
        - name: RATE_LIMIT_RPS
          value: {{ .Values.config.rateLimit.requestsPerSecond | quote }}
        - name: RATE_LIMIT_BURST
          value: {{ .Values.config.rateLimit.burstSize | quote }}
        {{- end }}
        - name: HEALTH_CHECK_ENABLED
          value: {{ .Values.config.healthCheck.enabled | quote }}
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        {{- with .Values.extraEnv }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- if .Values.livenessProbe }}
      livenessProbe:
        {{- toYaml .Values.livenessProbe | nindent 8 }}
      {{- end }}
      {{- if .Values.readinessProbe }}
      readinessProbe:
        {{- toYaml .Values.readinessProbe | nindent 8 }}
      {{- end }}
      {{- if .Values.startupProbe }}
      startupProbe:
        {{- toYaml .Values.startupProbe | nindent 8 }}
      {{- end }}
      resources:
        {{- toYaml .Values.resources | nindent 8 }}
      securityContext:
        {{- toYaml .Values.securityContext.container | nindent 8 }}
      volumeMounts:
        - name: config
          mountPath: {{ .Values.config.configPath }}
          readOnly: true
        {{- if .Values.envoyConfig.create }}
        - name: envoy-config
          mountPath: /etc/envoy
          readOnly: true
        {{- end }}
        - name: logs
          mountPath: {{ .Values.config.logPath }}
        {{- if or .Values.tls.createSecret .Values.certManager.enabled }}
        - name: tls-certs
          mountPath: {{ .Values.config.certPath }}
          readOnly: true
        {{- end }}
        - name: tmp
          mountPath: /tmp
        - name: run
          mountPath: /run
        {{- with .Values.extraVolumeMounts }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- with .Values.lifecycle }}
      lifecycle:
        {{- toYaml . | nindent 8 }}
      {{- end }}
  volumes:
    - name: config
      configMap:
        name: {{ include "marchproxy-egress.fullname" . }}
    {{- if .Values.envoyConfig.create }}
    - name: envoy-config
      configMap:
        name: {{ include "marchproxy-egress.fullname" . }}-envoy
    {{- end }}
    - name: logs
      {{- if .Values.persistence.enabled }}
      persistentVolumeClaim:
        claimName: {{ include "marchproxy-egress.fullname" . }}-logs
      {{- else }}
      emptyDir: {}
      {{- end }}
    {{- if or .Values.tls.createSecret .Values.certManager.enabled }}
    - name: tls-certs
      secret:
        secretName: {{ .Values.tls.secretName }}
        defaultMode: 0400
    {{- end }}
    - name: tmp
      emptyDir: {}
    - name: run
      emptyDir: {}
    {{- with .Values.extraVolumes }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- with .Values.nodeSelector }}
  nodeSelector:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.affinity }}
  affinity:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.tolerations }}
  tolerations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end -}}
