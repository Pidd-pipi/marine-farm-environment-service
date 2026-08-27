// enums.js — shared enum/constant definitions (frontend mirror of the
// backend domain/types.go). Keep the values in sync with the Go layer.
// Backend source of truth: domain/types.go.

export const ZoneStatus = {
  normal: 'normal',
  warning: 'warning',
  danger: 'danger',
  aerating: 'aerating',
  restored: 'restored',
};

export const ZONE_STATUS_LABEL = {
  normal: '正常运行',
  warning: '预警',
  danger: '危险',
  aerating: '应急增氧',
  restored: '恢复确认',
};

export const ZONE_STATUS_CLASS = {
  normal: 'status-normal',
  warning: 'status-warning',
  danger: 'status-danger',
  aerating: 'status-aerating',
  restored: 'status-restored',
};

export const WarningType = {
  do_low: 'do_low',
  temp_shock: 'temp_shock',
  ph_abnormal: 'ph_abnormal',
  ammonia_high: 'ammonia_high',
};

export const WARNING_TYPE_LABEL = {
  do_low: '溶解氧过低',
  temp_shock: '水温骤变',
  ph_abnormal: 'pH 异常',
  ammonia_high: '氨氮超标',
};

export const WarningStatus = {
  pending: 'pending',
  confirmed: 'confirmed',
  resolved: 'resolved',
};

export const WARNING_STATUS_LABEL = {
  pending: '待核实',
  confirmed: '已确认',
  resolved: '已解除',
};

export const WARNING_STATUS_CLASS = {
  pending: 'status-pending',
  confirmed: 'status-confirmed',
  resolved: 'status-resolved',
};

export const WarningLevel = {
  warning: 'warning',
  danger: 'danger',
};

export const WARNING_LEVEL_LABEL = {
  warning: '预警',
  danger: '危险',
};

export const AeratorStatus = {
  stopped: 'stopped',
  starting: 'starting',
  running: 'running',
  stopping: 'stopping',
  fault: 'fault',
};

export const AERATOR_STATUS_LABEL = {
  stopped: '已停止',
  starting: '启动中',
  running: '运行中',
  stopping: '停止中',
  fault: '故障',
};

export const AERATOR_STATUS_CLASS = {
  stopped: 'status-stopped',
  starting: 'status-starting',
  running: 'status-running',
  stopping: 'status-stopping',
  fault: 'status-fault',
};

export const AERATION_ACTION_LABEL = {
  start: '启动增氧',
  stop: '停止增氧',
};

export const FEEDBACK_LABEL = {
  none: '待反馈',
  acknowledged: '已应答',
  started: '已启动',
  stopped: '已停止',
  fault: '故障',
  timeout: '超时',
};

export const BuoyStatus = {
  active: 'active',
  offline: 'offline',
  maintenance: 'maintenance',
};

export const BUOY_STATUS_LABEL = {
  active: '在线',
  offline: '离线',
  maintenance: '维护中',
};
