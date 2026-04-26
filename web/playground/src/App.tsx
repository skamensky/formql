import React from 'react';
import ReactDOM from 'react-dom/client';
import { formql } from './realData';
import IDEArtboard from './IDEArtboard';
import ReportArtboard from './ReportArtboard';
import type { CompileMode } from './types';

function PlaygroundApp(): React.ReactElement {
  const [ready, setReady] = React.useState(false);
  const [initError, setInitError] = React.useState<string | null>(null);
  const [mode, setMode] = React.useState<CompileMode>('formula');

  React.useEffect(() => {
    formql.init()
      .then(() => setReady(true))
      .catch((e: unknown) => setInitError(e instanceof Error ? e.message : 'failed to initialize'));
  }, []);

  if (initError) {
    return <div className="load-screen err">Error: {initError}</div>;
  }
  if (!ready) {
    return <div className="load-screen">loading wasm…</div>;
  }

  if (mode === 'document') {
    return <ReportArtboard onMode={setMode} />;
  }
  return <IDEArtboard showRaw={false} onMode={setMode} />;
}

const rootEl = document.getElementById('root')!;
ReactDOM.createRoot(rootEl).render(<PlaygroundApp />);
