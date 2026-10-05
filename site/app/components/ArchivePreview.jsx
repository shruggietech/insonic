// SPDX-License-Identifier: Apache-2.0
const amplitudes = [8, 15, 9, 22, 32, 18, 26, 40, 28, 17, 9, 14, 22, 35, 44, 28, 19, 12, 18, 27, 37, 24, 17, 9, 12, 21, 30, 18, 8, 15, 26, 40, 29, 19, 10, 14, 23, 34, 25, 13, 8, 19, 28, 18, 12, 7];

export default function ArchivePreview() {
  return (
    <div className="archive-preview" aria-label="Illustrative source-linked voice archive">
      <div className="archive-heading"><span className="archive-dot" aria-hidden="true" /><strong>Voice archive</strong><span>Example data</span></div>
      <div className="archive-source"><span className="source-format">WAV</span><div><strong>session.wav</strong><small>Original audio · 2 speakers</small></div><span className="source-duration">00:32</span></div>
      <div className="waveform" aria-label="Illustrative audio waveform with speaker regions">
        <svg viewBox="0 0 480 112" role="img" aria-label="Two speaker regions in the original recording">
          <rect className="wave-region wave-region-first" x="0" y="0" width="234" height="112" rx="8" />
          <rect className="wave-region wave-region-second" x="244" y="0" width="236" height="112" rx="8" />
          {amplitudes.map((height, index) => <line className={index < 23 ? 'wave-first' : 'wave-second'} key={index} x1={10 + index * 10} x2={10 + index * 10} y1={56 - height} y2={56 + height} />)}
          <line className="wave-playhead" x1="120" x2="120" y1="6" y2="106" />
        </svg>
        <div className="wave-labels"><span>Speaker 01</span><span>Speaker 02</span></div>
      </div>
      <div className="archive-tabs" role="tablist" aria-label="Archive preview view">
        <button type="button" role="tab" id="preview-transcript-tab" aria-controls="preview-transcript" aria-selected="true" data-preview-tab="transcript">Transcript</button>
        <button type="button" role="tab" id="preview-connections-tab" aria-controls="preview-connections" aria-selected="false" tabIndex="-1" data-preview-tab="connections">Connections</button>
        <button type="button" role="tab" id="preview-source-tab" aria-controls="preview-source" aria-selected="false" tabIndex="-1" data-preview-tab="source">Source</button>
      </div>
      <div className="archive-panels">
      <div className="archive-panel" id="preview-transcript" role="tabpanel" aria-labelledby="preview-transcript-tab" data-preview-panel="transcript" tabIndex="0">
        <div className="preview-segment"><div><span className="speaker-chip">Speaker 01</span><time>00:08.200 – 00:12.600</time></div><p>“Keep the original recording.”</p><small>Source → session.wav · segment 01</small></div>
        <div className="preview-segment"><div><span className="speaker-chip speaker-second">Speaker 02</span><time>00:18.400 – 00:23.100</time></div><p>“Every segment stays linked to its source.”</p><small>Source → session.wav · segment 02</small></div>
      </div>
      <div className="archive-panel" id="preview-connections" role="tabpanel" aria-labelledby="preview-connections-tab" data-preview-panel="connections" tabIndex="0" hidden inert>
        <svg className="preview-graph" viewBox="0 0 460 220" role="img" aria-label="session.wav contains two source-timed segments attributed to Speaker 01 and Speaker 02">
          <path d="M230 46 L110 104 M230 46 L350 104 M110 134 L110 176 M350 134 L350 176" />
          <g><rect x="157" y="12" width="146" height="42" rx="8" /><text x="230" y="38">session.wav</text></g>
          <g><rect x="40" y="98" width="140" height="40" rx="8" /><text x="110" y="123">00:08.200</text></g>
          <g><rect x="280" y="98" width="140" height="40" rx="8" /><text x="350" y="123">00:18.400</text></g>
          <g><rect x="40" y="176" width="140" height="40" rx="8" /><text x="110" y="201">Speaker 01</text></g>
          <g><rect x="280" y="176" width="140" height="40" rx="8" /><text x="350" y="201">Speaker 02</text></g>
        </svg>
        <p className="preview-explanation">Follow either speaker through their segment to the original audio.</p>
      </div>
      <div className="archive-panel" id="preview-source" role="tabpanel" aria-labelledby="preview-source-tab" data-preview-panel="source" tabIndex="0" hidden inert>
        <dl className="source-facts"><div><dt>Original</dt><dd>session.wav</dd></div><div><dt>Metadata capture</dt><dd>Before processing</dd></div><div><dt>Subtitle layer</dt><dd>Cueson</dd></div><div><dt>Derived audio</dt><dd>Mapped to source time</dd></div><div><dt>Speaker corpus</dt><dd>Linked across the library</dd></div></dl>
      </div>
      </div>
      <div className="archive-footnote"><span>Original → segments → speakers</span><span>Source-linked</span></div>
    </div>
  );
}
