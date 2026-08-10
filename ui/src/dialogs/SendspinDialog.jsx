import React, { useCallback, useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import {
  Button,
  Dialog,
  IconButton,
  List,
  ListItem,
  ListItemSecondaryAction,
  ListItemText,
  Slider,
  Typography,
  makeStyles,
  Box,
  Chip,
} from '@material-ui/core'
import {
  MdPause,
  MdPlayArrow,
  MdSkipNext,
  MdSkipPrevious,
  MdRefresh,
  MdContentCopy,
  MdSpeaker,
} from 'react-icons/md'
import { useNotify, useTranslate } from 'react-admin'
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const useStyles = makeStyles((theme) => ({
  section: {
    marginBottom: theme.spacing(2),
  },
  urlRow: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    marginTop: theme.spacing(1),
  },
  url: {
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    wordBreak: 'break-all',
    flex: 1,
  },
  controls: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    marginTop: theme.spacing(1),
  },
  empty: {
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
  },
  chip: {
    marginLeft: theme.spacing(1),
  },
  gain: {
    paddingLeft: theme.spacing(1),
    paddingRight: theme.spacing(2),
  },
}))

const formatNowPlaying = (np, translate) => {
  if (!np || (!np.title && !np.artist)) {
    return translate('menu.sendspin.idle')
  }
  if (np.artist && np.title) {
    return `${np.artist} – ${np.title}`
  }
  return np.title || np.artist
}

export const SendspinDialog = ({ open, onClose }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(false)

  const loadStatus = useCallback(() => {
    setLoading(true)
    return httpClient(`${REST_URL}/sendspin`)
      .then(({ json }) => setStatus(json))
      .catch((e) => {
        notify('menu.sendspin.loadError', 'warning')
        console.error(e)
      })
      .finally(() => setLoading(false))
  }, [notify])

  useEffect(() => {
    if (!open) {
      return undefined
    }
    loadStatus()
    const id = setInterval(loadStatus, 4000)
    return () => clearInterval(id)
  }, [open, loadStatus])

  const sendCommand = (command) => {
    httpClient(`${REST_URL}/sendspin/command`, {
      method: 'POST',
      body: JSON.stringify({ command }),
    })
      .then(({ json }) => setStatus(json))
      .catch(() => notify('menu.sendspin.commandError', 'warning'))
  }

  const setGain = (_, value) => {
    const gain = Array.isArray(value) ? value[0] : value
    setStatus((prev) => (prev ? { ...prev, gain } : prev))
  }

  const commitGain = (_, value) => {
    const gain = Array.isArray(value) ? value[0] : value
    httpClient(`${REST_URL}/sendspin/gain`, {
      method: 'POST',
      body: JSON.stringify({ gain }),
    })
      .then(({ json }) => setStatus(json))
      .catch(() => notify('menu.sendspin.commandError', 'warning'))
  }

  const copyUrl = () => {
    if (!status?.connectionUrl) {
      return
    }
    navigator.clipboard
      .writeText(status.connectionUrl)
      .then(() => notify('menu.sendspin.urlCopied', 'info'))
      .catch(() => notify('menu.sendspin.copyError', 'warning'))
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      aria-labelledby="sendspin-dialog-title"
    >
      <DialogTitle id="sendspin-dialog-title" onClose={onClose}>
        {translate('menu.sendspin.name')}
      </DialogTitle>
      <DialogContent dividers>
        {status && (
          <>
            <Box className={classes.section}>
              <Typography variant="subtitle2">
                {translate('menu.sendspin.server')}
                <Chip
                  size="small"
                  className={classes.chip}
                  color={status.running ? 'primary' : 'default'}
                  label={
                    status.running
                      ? translate('menu.sendspin.running')
                      : translate('menu.sendspin.stopped')
                  }
                />
              </Typography>
              <Typography variant="body2">
                {status.serverName} · :{status.port}
              </Typography>
              <div className={classes.urlRow}>
                <Typography className={classes.url}>
                  {status.connectionUrl}
                </Typography>
                <IconButton size="small" onClick={copyUrl} title={translate('menu.sendspin.copyUrl')}>
                  <MdContentCopy size={18} />
                </IconButton>
              </div>
            </Box>

            <Box className={classes.section}>
              <Typography variant="subtitle2">
                {translate('menu.sendspin.nowPlaying')}
              </Typography>
              <Typography variant="body2">
                {formatNowPlaying(status.nowPlaying, translate)}
                {status.playing ? ` (${translate('menu.sendspin.playing')})` : ''}
              </Typography>
              <div className={classes.controls}>
                <IconButton
                  onClick={() => sendCommand('previous')}
                  title={translate('menu.sendspin.previous')}
                >
                  <MdSkipPrevious size={22} />
                </IconButton>
                {status.playing ? (
                  <IconButton
                    onClick={() => sendCommand('pause')}
                    title={translate('menu.sendspin.pause')}
                    color="primary"
                  >
                    <MdPause size={22} />
                  </IconButton>
                ) : (
                  <IconButton
                    onClick={() => sendCommand('play')}
                    title={translate('menu.sendspin.play')}
                    color="primary"
                  >
                    <MdPlayArrow size={22} />
                  </IconButton>
                )}
                <IconButton
                  onClick={() => sendCommand('next')}
                  title={translate('menu.sendspin.next')}
                >
                  <MdSkipNext size={22} />
                </IconButton>
                <IconButton
                  onClick={loadStatus}
                  disabled={loading}
                  title={translate('menu.sendspin.refresh')}
                >
                  <MdRefresh size={20} />
                </IconButton>
              </div>
              <Typography variant="caption" display="block" gutterBottom>
                {translate('menu.sendspin.gain')}
              </Typography>
              <Slider
                className={classes.gain}
                value={typeof status.gain === 'number' ? status.gain : 1}
                min={0}
                max={1}
                step={0.01}
                onChange={setGain}
                onChangeCommitted={commitGain}
                valueLabelDisplay="auto"
                valueLabelFormat={(v) => `${Math.round(v * 100)}%`}
              />
            </Box>

            <Box className={classes.section}>
              <Typography variant="subtitle2" gutterBottom>
                {translate('menu.sendspin.devices', {
                  smart_count: status.clients?.length || 0,
                })}
              </Typography>
              {!status.clients?.length ? (
                <Typography className={classes.empty} variant="body2">
                  {translate('menu.sendspin.noDevices')}
                </Typography>
              ) : (
                <List dense>
                  {status.clients.map((client) => (
                    <ListItem key={client.id} divider>
                      <MdSpeaker size={20} style={{ marginRight: 12 }} />
                      <ListItemText
                        primary={client.name || client.id}
                        secondary={`${client.state || 'unknown'}${
                          client.codec ? ` · ${client.codec}` : ''
                        }${client.muted ? ` · ${translate('menu.sendspin.muted')}` : ''}`}
                      />
                      <ListItemSecondaryAction>
                        <Typography variant="caption">
                          {typeof client.volume === 'number'
                            ? `${client.volume}%`
                            : ''}
                        </Typography>
                      </ListItemSecondaryAction>
                    </ListItem>
                  ))}
                </List>
              )}
            </Box>
          </>
        )}
        {!status && !loading && (
          <Typography className={classes.empty}>
            {translate('menu.sendspin.loadError')}
          </Typography>
        )}
        <Button onClick={onClose} color="primary">
          {translate('ra.action.close')}
        </Button>
      </DialogContent>
    </Dialog>
  )
}

SendspinDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}
