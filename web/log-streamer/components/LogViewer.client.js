"use client";
import React, { useState, useEffect, useRef } from 'react';
import { Button, List, ListItem, Paper, Typography, Box, IconButton, TextField } from '@mui/material';
import PauseIcon from '@mui/icons-material/Pause';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import AnsiToHtml from 'ansi-to-html';

const convert = new AnsiToHtml();

const LogViewer = () => {
  const [ logMessages, setLogMessages ] = useState([]);
  const [ logQueue, setLogQueue ] = useState([]);
  const [ filter, setFilter ] = useState('');
  const [ isPaused, setIsPaused ] = useState(false);
  const [ maxLogLines, setMaxLogLines ] = useState(200);
  const [ ws, setWs ] = useState(null);

  const bottomListRef = useRef(null);
  const lastConnect = useRef(0);

  const formatLogMessage = (msg) => {
    // 将 ANSI 代码转换为 HTML
    return convert.toHtml(msg);
  };

  useEffect(() => {
    const connectWebSocket = () => {
      const wsUrl = process.env.NEXT_PUBLIC_BACKEND_URL;
      console.log(`Connecting to WebSocket at ${ wsUrl }...`)
      const ws = new WebSocket(wsUrl);
      setWs(ws);

      ws.onopen = () => {
        console.log("WebSocket connected.");
      }

      ws.onmessage = (event) => {
        setLogQueue((prevQueue) => {
          const updatedQueue = [ ...prevQueue, event.data ];
          // 如果队列长度超过最大行数，从头部删除
          if (updatedQueue.length > maxLogLines) {
            updatedQueue.shift();
          }
          return updatedQueue;
        });
      };

      ws.onclose = () => {
        console.log("WebSocket disconnected. Attempting to reconnect...");
        setWs(null);
      };

      ws.onerror = (err) => {
        console.error("WebSocket error:", err);
        ws.close();
      }
    };

    const now = Date.now();
    if (!ws && (lastConnect.current + 1000 < now || !lastConnect.current)) {
      connectWebSocket();
      lastConnect.current = now;
    }

    return () => {
      if (ws) ws.close();
    };
  }, [ ws, maxLogLines ]);

  const scrollToBottom = () => {
    bottomListRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const handlePauseClick = () => {
    setIsPaused(!isPaused);
  };

  useEffect(() => {
    if (!isPaused && logQueue.length > 0) {
      setLogMessages((prevMessages) => {
        const updatedMessages = [ ...prevMessages, ...logQueue ];
        // 如果队列长度超过最大行数，从头部删除
        if (updatedMessages.length > maxLogLines) {
          updatedMessages.splice(0, updatedMessages.length - maxLogLines);
        }
        return updatedMessages;
      });
      setLogQueue([]); // 清空队列
      scrollToBottom(); // 滚动到底部
    }
  }, [ logQueue, isPaused, maxLogLines ]);

  const handleFilterChange = (e) => {
    setFilter(e.target.value);
  };

  const handleMaxLogLinesChange = (e) => {
    setMaxLogLines(e.target.value === '' ? 0 : Math.max(0, parseInt(e.target.value, 10)));
  };

  const clearLogs = () => {
    setLogMessages([]);
    setCachedMessages([]);
  };

  const togglePause = () => {
    setIsPaused(!isPaused);
  };

  const filteredLogs = logMessages.filter((msg) => msg.includes(filter));

  return (
    <Box className="flex flex-col items-center justify-center w-full px-4">
      <Typography variant="h4" component="h1" gutterBottom align="center">
        Artela Realtime Log Viewer
        <span className={ `status-indicator ${ !ws ? 'disconnected' : 'connected' }` }></span>
      </Typography>
      <Box style={ { marginBottom: '10px', display: 'flex', gap: '10px', alignItems: 'center' } }>
        <TextField
          label="Filter logs"
          variant="outlined"
          value={ filter }
          onChange={ handleFilterChange }
          size="small"
        />
        <TextField
          label="Max log lines"
          variant="outlined"
          value={ maxLogLines }
          onChange={ handleMaxLogLinesChange }
          size="small"
          type="number"
        />
        <Button variant="contained" color="primary" onClick={ clearLogs }>Clear</Button>
        <IconButton onClick={ handlePauseClick }>
          { isPaused ? <PlayArrowIcon/> : <PauseIcon/> }
        </IconButton>
      </Box>
      <Paper className="w-full overflow-y-auto mt-2" style={ { height: 'calc(100vh - 150px)' } }>
        <List className="divide-y divide-gray-200">
          { filteredLogs.map((msg, index) => (
            <ListItem key={ index } className="pr-4">
              <div className={ 'break-all' } dangerouslySetInnerHTML={ { __html: formatLogMessage(msg) } }/>
            </ListItem>
          )) }
          <div ref={ bottomListRef }/>
        </List>
      </Paper>
    </Box>
  );
};

export default LogViewer;
