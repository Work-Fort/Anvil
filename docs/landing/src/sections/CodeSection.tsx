import { useRef, useLayoutEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

interface CodeSectionProps {
  className?: string;
}

const codeLines = [
  { content: '$ # Download latest Firecracker kernel', type: 'comment' },
  { content: '$ wget https://github.com/get-virgil/cracker-barrel/releases/latest/download/vmlinux-6.18.9-x86_64.xz', type: 'keyword' },
  { content: '', type: 'empty' },
  { content: '$ # Verify cryptographic signature', type: 'comment' },
  { content: '$ wget https://github.com/get-virgil/cracker-barrel/releases/latest/download/SHA256SUMS.asc', type: 'keyword' },
  { content: '$ gpg --verify SHA256SUMS.asc', type: 'keyword' },
  { content: 'gpg: Good signature from "Cracker Barrel"', type: 'string' },
  { content: '', type: 'empty' },
  { content: '$ # Decompress and boot', type: 'comment' },
  { content: '$ xz -d vmlinux-6.18.9-x86_64.xz', type: 'keyword' },
  { content: '$ firecracker --kernel-image vmlinux-6.18.9-x86_64 --config vm.json', type: 'keyword' },
  { content: '', type: 'empty' },
  { content: '[    0.000000] Linux version 6.18.9 (cracker-barrel)', type: 'string' },
  { content: '[    0.123456] Freeing unused kernel memory: 2048K', type: 'string' },
  { content: '[    0.234567] Run /sbin/init as init process', type: 'string' },
  { content: '', type: 'empty' },
  { content: '✓ VM booted successfully in 0.1s', type: 'string' },
];

export default function CodeSection({ className = '' }: CodeSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const linesRef = useRef<(HTMLDivElement | null)[]>([]);
  const captionRef = useRef<HTMLParagraphElement>(null);
  const bgRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const ctx = gsap.context(() => {
      const scrollTl = gsap.timeline({
        scrollTrigger: {
          trigger: section,
          start: 'top top',
          end: '+=130%',
          pin: true,
          scrub: 0.6,
        }
      });

      // ENTRANCE (0-30%)
      // Background
      scrollTl.fromTo(bgRef.current,
        { scale: 1.08 },
        { scale: 1, ease: 'none' },
        0
      );

      // Panel entrance
      scrollTl.fromTo(panelRef.current,
        { y: '90vh', scale: 0.98, opacity: 0 },
        { y: 0, scale: 1, opacity: 1, ease: 'power1.out' },
        0
      );

      // Code lines entrance (staggered)
      linesRef.current.forEach((line, i) => {
        if (!line) return;
        scrollTl.fromTo(line,
          { x: -40, opacity: 0 },
          { x: 0, opacity: 1, ease: 'power1.out' },
          0.10 + i * 0.015
        );
      });

      // Caption entrance
      scrollTl.fromTo(captionRef.current,
        { y: 20, opacity: 0 },
        { y: 0, opacity: 1, ease: 'power1.out' },
        0.25
      );

      // SETTLE (30-70%): Hold

      // EXIT (70-100%)
      scrollTl.fromTo(panelRef.current,
        { y: 0, opacity: 1 },
        { y: '-35vh', opacity: 0, ease: 'power2.in' },
        0.70
      );

      linesRef.current.forEach((line) => {
        if (!line) return;
        scrollTl.fromTo(line,
          { x: 0, opacity: 1 },
          { x: 20, opacity: 0, ease: 'power2.in' },
          0.72
        );
      });

      scrollTl.fromTo(bgRef.current,
        { scale: 1 },
        { scale: 1.05, ease: 'none' },
        0.70
      );

    }, section);

    return () => ctx.revert();
  }, []);

  const getLineClass = (type: string) => {
    switch (type) {
      case 'comment': return 'code-comment';
      case 'string': return 'code-string';
      case 'keyword': return 'code-keyword';
      default: return '';
    }
  };

  return (
    <section 
      ref={sectionRef}
      className={`relative w-full h-screen overflow-hidden ${className}`}
    >
      {/* Background Image */}
      <div 
        ref={bgRef}
        className="absolute inset-0 w-full h-full"
      >
        <img 
          src="./chrome_code_bg.jpg" 
          alt="Chrome Interior"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-[#07070A]/70" />
      </div>

      {/* Code Panel */}
      <div className="relative z-10 flex items-center justify-center h-full px-4 sm:px-6">
        <div 
          ref={panelRef}
          className="relative w-full max-w-[1040px] bg-[#141419] border border-white/[0.08] rounded-[22px] card-shadow overflow-hidden"
          style={{ height: 'min(62vh, 520px)' }}
        >
          {/* Panel Header */}
          <div className="h-11 sm:h-12 bg-white/[0.04] flex items-center px-4 border-b border-white/[0.06]">
            {/* Window dots */}
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-[#FF5F57]" />
              <div className="w-3 h-3 rounded-full bg-[#FEBC2E]" />
              <div className="w-3 h-3 rounded-full bg-[#28C840]" />
            </div>
            {/* Title */}
            <div className="flex-1 text-center">
              <span className="text-[#A7ACB8] text-xs sm:text-sm font-mono">terminal — getting-started.sh</span>
            </div>
            {/* Spacer for alignment */}
            <div className="w-14" />
          </div>

          {/* Code Content */}
          <div className="p-4 sm:p-6 overflow-auto h-[calc(100%-48px)]">
            <pre className="font-mono text-sm leading-relaxed">
              {codeLines.map((line, index) => (
                <div 
                  key={index}
                  ref={el => { linesRef.current[index] = el; }}
                  className="flex"
                >
                  <span className="text-[#6F7682] select-none w-8 text-right mr-4 flex-shrink-0">
                    {index + 1}
                  </span>
                  <span className={getLineClass(line.type)}>
                    {line.content || ' '}
                  </span>
                </div>
              ))}
            </pre>
          </div>
        </div>

        {/* Caption */}
        <p
          ref={captionRef}
          className="absolute bottom-[15vh] text-[#A7ACB8] text-sm sm:text-base text-center"
        >
          Download, verify, boot. Fresh kernels in seconds, not hours.
        </p>
      </div>
    </section>
  );
}
