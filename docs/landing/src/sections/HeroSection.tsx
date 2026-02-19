import { useEffect, useRef, useLayoutEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { BookOpen, Download } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface HeroSectionProps {
  className?: string;
}

export default function HeroSection({ className = '' }: HeroSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const neonSignRef = useRef<HTMLDivElement>(null);
  const taglineRef = useRef<HTMLDivElement>(null);
  const ctaRef = useRef<HTMLDivElement>(null);
  const microcopyRef = useRef<HTMLDivElement>(null);
  const chickenRef = useRef<HTMLImageElement>(null);
  const bgRef = useRef<HTMLDivElement>(null);

  // Auto-play entrance animation on load
  useEffect(() => {
    const ctx = gsap.context(() => {
      const tl = gsap.timeline({ defaults: { ease: 'power2.out' } });

      // Background fade in
      tl.fromTo(bgRef.current,
        { opacity: 0 },
        { opacity: 1, duration: 0.6 }
      );

      // Neon sign entrance
      tl.fromTo(neonSignRef.current,
        { scale: 1.08, opacity: 0 },
        { scale: 1, opacity: 1, duration: 0.9, ease: 'power2.out' },
        '-=0.3'
      );

      // Cyborg chicken entrance
      tl.fromTo(chickenRef.current,
        { y: 40, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.7 },
        '-=0.5'
      );

      // Tagline + CTAs
      tl.fromTo(taglineRef.current,
        { y: 18, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.5 },
        '-=0.3'
      );

      tl.fromTo(ctaRef.current,
        { y: 18, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.5 },
        '-=0.3'
      );

      tl.fromTo(microcopyRef.current,
        { y: 18, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.5 },
        '-=0.3'
      );
    }, sectionRef);

    return () => ctx.revert();
  }, []);

  // Scroll-driven exit animation
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
          onLeaveBack: () => {
            // Reset all elements when scrolling back to top
            gsap.set([neonSignRef.current, taglineRef.current, ctaRef.current, microcopyRef.current, chickenRef.current], {
              opacity: 1, y: 0, x: 0
            });
            gsap.set(bgRef.current, { scale: 1 });
          }
        }
      });

      // ENTRANCE (0-30%): Hold - elements already visible from load animation
      // SETTLE (30-70%): Hold
      
      // EXIT (70-100%)
      scrollTl.fromTo(neonSignRef.current,
        { y: 0, opacity: 1 },
        { y: '-18vh', opacity: 0, ease: 'power2.in' },
        0.70
      );

      scrollTl.fromTo(chickenRef.current,
        { y: 0, opacity: 1 },
        { y: '-12vh', opacity: 0, ease: 'power2.in' },
        0.72
      );

      scrollTl.fromTo(taglineRef.current,
        { y: 0, opacity: 1 },
        { y: '-10vh', opacity: 0, ease: 'power2.in' },
        0.72
      );

      scrollTl.fromTo(ctaRef.current,
        { y: 0, opacity: 1 },
        { y: '-10vh', opacity: 0, ease: 'power2.in' },
        0.74
      );

      scrollTl.fromTo(microcopyRef.current,
        { y: 0, opacity: 1 },
        { y: '-10vh', opacity: 0, ease: 'power2.in' },
        0.76
      );

      scrollTl.fromTo(bgRef.current,
        { scale: 1 },
        { scale: 1.06, ease: 'none' },
        0.70
      );

    }, section);

    return () => ctx.revert();
  }, []);

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
          src="./hero_diner_bg.jpg" 
          alt="Cyberpunk Diner"
          className="w-full h-full object-cover"
        />
        {/* Dark overlay for text readability */}
        <div className="absolute inset-0 bg-gradient-to-b from-[#07070A]/60 via-[#07070A]/40 to-[#07070A]/80" />
      </div>

      {/* Content */}
      <div className="relative z-10 flex flex-col items-center justify-center h-full px-6">
        {/* Neon Sign */}
        <div 
          ref={neonSignRef}
          className="relative mb-4"
        >
          <h1 className="neon-sign text-[clamp(44px,8vw,84px)] font-black tracking-[0.02em] text-[#B9FF2C] neon-glow uppercase text-center leading-[0.92]">
            CRACKER BARREL
          </h1>
          <div className="absolute -bottom-2 left-1/2 -translate-x-1/2 w-3/4 h-px bg-gradient-to-r from-transparent via-[#B9FF2C]/50 to-transparent" />
        </div>

        {/* Cyborg Chicken Chef */}
        <img 
          ref={chickenRef}
          src="./cyborg_chicken_chef.png" 
          alt="Cyborg Chicken Chef"
          className="w-[clamp(120px,20vw,200px)] h-auto object-contain mb-6 drop-shadow-2xl"
        />

        {/* Tagline */}
        <div ref={taglineRef} className="text-center mb-8">
          <p className="text-[clamp(18px,2.5vw,28px)] text-[#F4F6FA] font-medium tracking-wide">
            Get Firecracker kernels like you're ordering breakfast.
          </p>
        </div>

        {/* CTA Buttons */}
        <div ref={ctaRef} className="flex flex-wrap items-center justify-center gap-4 mb-8">
          <a
            href="#install"
            className="flex items-center gap-2 bg-[#B9FF2C] text-[#07070A] px-6 py-3 rounded-full font-semibold hover:brightness-110 transition-all hover:-translate-y-0.5 hover:neon-box-glow"
          >
            <Download className="w-5 h-5" />
            Download
          </a>
          <a
            href="./docs/index.html"
            className="flex items-center gap-2 border border-white/20 text-[#F4F6FA] px-6 py-3 rounded-full font-medium hover:border-[#B9FF2C]/50 hover:text-[#B9FF2C] transition-all hover:-translate-y-0.5"
          >
            <BookOpen className="w-5 h-5" />
            Read the Docs
          </a>
        </div>

        {/* Microcopy */}
        <div ref={microcopyRef} className="text-center max-w-xl">
          <p className="text-[#A7ACB8] text-sm sm:text-base">
            Hot, fresh kernels served daily. Cryptographically seasoned. x86_64 or CISC-free (ARM64).
          </p>
        </div>
      </div>
    </section>
  );
}
