import { useRef, useLayoutEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { Package, GitPullRequest, Rocket, ArrowRight } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface DetailsSectionProps {
  className?: string;
}

const bullets = [
  {
    icon: Package,
    text: 'Browse our daily specials—fresh stable kernels, built and ready to go.'
  },
  {
    icon: GitPullRequest,
    text: "Don't see what you want? Request any kernel version via GitHub issue."
  },
  {
    icon: Rocket,
    text: "We'll build it, sign it, and serve it hot. 30 minutes or less, or it's on GitHub."
  }
];

export default function DetailsSection({ className = '' }: DetailsSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const mediaRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const eyebrowRef = useRef<HTMLSpanElement>(null);
  const headlineRef = useRef<HTMLHeadingElement>(null);
  const bulletsRef = useRef<(HTMLDivElement | null)[]>([]);
  const ctaRef = useRef<HTMLAnchorElement>(null);
  const bgRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const ctx = gsap.context(() => {
      const scrollTl = gsap.timeline({
        scrollTrigger: {
          trigger: section,
          start: 'top top',
          end: '+=140%',
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

      // Main card entrance
      scrollTl.fromTo(cardRef.current,
        { x: '60vw', rotate: 2, opacity: 0 },
        { x: 0, rotate: 0, opacity: 1, ease: 'power1.out' },
        0
      );

      // Media inside card
      scrollTl.fromTo(mediaRef.current,
        { scale: 1.10, x: '-6%' },
        { scale: 1, x: '0%', ease: 'power1.out' },
        0
      );

      // Content elements (staggered)
      scrollTl.fromTo(eyebrowRef.current,
        { y: 30, opacity: 0 },
        { y: 0, opacity: 1, ease: 'power1.out' },
        0.10
      );

      scrollTl.fromTo(headlineRef.current,
        { y: 30, opacity: 0 },
        { y: 0, opacity: 1, ease: 'power1.out' },
        0.14
      );

      bulletsRef.current.forEach((bullet, i) => {
        if (!bullet) return;
        scrollTl.fromTo(bullet,
          { x: 40, opacity: 0 },
          { x: 0, opacity: 1, ease: 'power1.out' },
          0.18 + i * 0.04
        );
      });

      scrollTl.fromTo(ctaRef.current,
        { y: 20, opacity: 0 },
        { y: 0, opacity: 1, ease: 'power1.out' },
        0.28
      );

      // SETTLE (30-70%): Hold

      // EXIT (70-100%)
      scrollTl.fromTo(cardRef.current,
        { x: 0, rotate: 0, opacity: 1 },
        { x: '-50vw', rotate: -2, opacity: 0, ease: 'power2.in' },
        0.70
      );

      scrollTl.fromTo(mediaRef.current,
        { scale: 1, x: '0%' },
        { scale: 1.06, x: '4%', ease: 'power2.in' },
        0.70
      );

      scrollTl.fromTo(contentRef.current,
        { y: 0, opacity: 1 },
        { y: '-10vh', opacity: 0, ease: 'power2.in' },
        0.72
      );

      scrollTl.fromTo(bgRef.current,
        { scale: 1 },
        { scale: 1.05, ease: 'none' },
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
          src="./detail_counter_bg.jpg" 
          alt="Diner Counter"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-[#07070A]/60" />
      </div>

      {/* Feature Card */}
      <div className="relative z-10 flex items-center justify-center h-full px-4 sm:px-6">
        <div 
          ref={cardRef}
          className="relative w-full max-w-[1220px] bg-[#141419] border border-white/[0.08] rounded-[22px] card-shadow overflow-hidden"
          style={{ height: 'min(70vh, 580px)' }}
        >
          <div className="flex flex-col lg:flex-row h-full">
            {/* Left Media */}
            <div 
              ref={mediaRef}
              className="relative w-full lg:w-[55%] h-48 lg:h-full overflow-hidden"
            >
              <img 
                src="./detail_feature_media.jpg" 
                alt="Kitchen Window"
                className="w-full h-full object-cover"
              />
              <div className="absolute inset-0 bg-gradient-to-r from-transparent to-[#141419]/80 hidden lg:block" />
              <div className="absolute inset-0 bg-gradient-to-t from-[#141419] to-transparent lg:hidden" />
            </div>

            {/* Right Content */}
            <div 
              ref={contentRef}
              className="flex-1 p-6 sm:p-8 lg:p-10 flex flex-col justify-center"
            >
              <span
                ref={eyebrowRef}
                className="text-[#B9FF2C] text-sm font-semibold tracking-wider uppercase mb-3"
              >
                Custom Orders Welcome
              </span>
              
              <h2 
                ref={headlineRef}
                className="text-[clamp(28px,4vw,44px)] font-black text-[#F4F6FA] leading-tight mb-6"
              >
                Order up. Boot up.
              </h2>

              {/* Bullets */}
              <div className="space-y-4 mb-8">
                {bullets.map((bullet, index) => (
                  <div
                    key={index}
                    ref={el => { bulletsRef.current[index] = el; }}
                    className="flex items-start gap-4"
                  >
                    <div className="flex-shrink-0 w-8 h-8 rounded-full bg-[#B9FF2C]/10 flex items-center justify-center mt-0.5">
                      <bullet.icon className="w-4 h-4 text-[#B9FF2C]" />
                    </div>
                    <p className="text-[#A7ACB8] text-sm sm:text-base leading-relaxed">
                      {bullet.text}
                    </p>
                  </div>
                ))}
              </div>

              {/* CTA */}
              <a
                ref={ctaRef}
                href="./docs/user-guide/github-workflow/requesting-builds.html"
                className="inline-flex items-center gap-2 bg-[#B9FF2C] text-[#07070A] px-6 py-3 rounded-full font-semibold w-fit hover:brightness-110 transition-all hover:-translate-y-0.5 hover:neon-box-glow"
              >
                Request a build
                <ArrowRight className="w-4 h-4" />
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
