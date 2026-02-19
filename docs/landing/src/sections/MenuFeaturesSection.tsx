import { useRef, useLayoutEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { ArrowRight } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface MenuFeaturesSectionProps {
  className?: string;
}

const menuItems = [
  {
    id: 1,
    title: 'Kernel Panini',
    description: 'Pre-built, pressed, and ready. Firecracker-optimized. Just heat and serve.',
    link: 'Get kernels',
    href: './docs/getting-started/github-releases.html',
    image: './menu_plate_1_kernel.jpg'
  },
  {
    id: 2,
    title: 'Security Seasoning',
    description: 'Cryptographically signed to perfection. PGP verification on every order.',
    link: 'See the proof',
    href: './docs/reference/security-model.html',
    image: './menu_plate_2_init.jpg'
  },
  {
    id: 3,
    title: 'The Daily Catch',
    description: 'Fresh kernel builds every morning. Latest stable. Never frozen.',
    link: 'How it works',
    href: './docs/getting-started/github-releases.html',
    image: './menu_plate_3_rootfs.jpg'
  }
];

export default function MenuFeaturesSection({ className = '' }: MenuFeaturesSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const boardRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const platesRef = useRef<(HTMLDivElement | null)[]>([]);
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
      // Background parallax
      scrollTl.fromTo(bgRef.current,
        { scale: 1.08 },
        { scale: 1, ease: 'none' },
        0
      );

      // Menu board entrance
      scrollTl.fromTo(boardRef.current,
        { y: '100vh', scale: 0.96, opacity: 0 },
        { y: 0, scale: 1, opacity: 1, ease: 'power1.out' },
        0
      );

      // Title entrance
      scrollTl.fromTo(titleRef.current,
        { y: -40, opacity: 0 },
        { y: 0, opacity: 1, ease: 'power1.out' },
        0.05
      );

      // Plates entrance (staggered)
      platesRef.current.forEach((plate, i) => {
        if (!plate) return;
        scrollTl.fromTo(plate,
          { y: '60vh', opacity: 0 },
          { y: 0, opacity: 1, ease: 'power1.out' },
          0.10 + i * 0.06
        );
      });

      // SETTLE (30-70%): Hold - no animation

      // EXIT (70-100%)
      scrollTl.fromTo(boardRef.current,
        { y: 0, opacity: 1 },
        { y: '-40vh', opacity: 0, ease: 'power2.in' },
        0.70
      );

      scrollTl.fromTo(titleRef.current,
        { y: 0, opacity: 1 },
        { y: '-10vh', opacity: 0, ease: 'power2.in' },
        0.70
      );

      platesRef.current.forEach((plate) => {
        if (!plate) return;
        scrollTl.fromTo(plate,
          { y: 0, opacity: 1 },
          { y: '-18vh', opacity: 0, ease: 'power2.in' },
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
          src="./menu_diner_bg.jpg" 
          alt="Diner Counter"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-[#07070A]/70" />
      </div>

      {/* Menu Board */}
      <div className="relative z-10 flex items-center justify-center h-full px-4 sm:px-6">
        <div 
          ref={boardRef}
          className="relative w-full max-w-[1180px] bg-[#141419] border border-white/[0.08] rounded-[22px] card-shadow p-6 sm:p-8"
          style={{ height: 'min(72vh, 600px)' }}
        >
          {/* Board Title */}
          <h2 
            ref={titleRef}
            className="text-center text-[clamp(24px,4vw,40px)] font-black text-[#F4F6FA] tracking-wider uppercase mb-6 sm:mb-8"
          >
            <span className="text-[#B9FF2C]">★</span> Today's Menu <span className="text-[#B9FF2C]">★</span>
          </h2>

          {/* Plates Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 sm:gap-6 h-[calc(100%-80px)]">
            {menuItems.map((item, index) => (
              <div
                key={item.id}
                ref={el => { platesRef.current[index] = el; }}
                className="group relative bg-[#07070A] border border-white/[0.10] rounded-[14px] overflow-hidden flex flex-col hover:border-[#B9FF2C]/30 transition-all duration-300 hover:-translate-y-1"
              >
                {/* Image */}
                <div className="relative h-32 sm:h-40 overflow-hidden">
                  <img 
                    src={item.image} 
                    alt={item.title}
                    className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-[#07070A] to-transparent" />
                </div>

                {/* Content */}
                <div className="flex-1 p-4 sm:p-5 flex flex-col">
                  <h3 className="text-[#F4F6FA] font-bold text-lg sm:text-xl mb-2 group-hover:text-[#B9FF2C] transition-colors">
                    {item.title}
                  </h3>
                  <p className="text-[#A7ACB8] text-sm leading-relaxed flex-1">
                    {item.description}
                  </p>
                  <a
                    href={item.href}
                    className="inline-flex items-center gap-1 text-[#B9FF2C] text-sm font-medium mt-3 hover:gap-2 transition-all"
                  >
                    {item.link}
                    <ArrowRight className="w-4 h-4" />
                  </a>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
